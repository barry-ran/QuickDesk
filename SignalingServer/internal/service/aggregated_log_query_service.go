package service

import (
	"context"
	"strings"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// AggregatedLogQuery is one search over the shipped logs of every instance.
// It is the cross-instance counterpart of LogQuery: that one reads files on the
// machine that answered the request, this one reads the shared table every
// instance ships into.
type AggregatedLogQuery struct {
	// Query is a case-insensitive substring match over the whole line.
	Query string
	// Match is a PostgreSQL full-text expression (`websearch_to_tsquery`
	// syntax): plain words are ANDed, "quoted phrases" are phrases, OR is
	// supported and a leading `-` negates. It is a second, separate mode rather
	// than a change to Query because the two answer different questions -
	// substring finds a request id, full text finds "connection refused"
	// without matching every line that contains both words apart.
	Match string
	// Component matches the structured `component=...` token, if present.
	Component string
	Since     *time.Time
	Until     *time.Time
	Limit     int
	// Instance narrows to one instance id; empty means every instance.
	Instance string
	// Source narrows to one logical source (`signaling.log`); empty means all.
	Source string
}

// AggregatedLogEntry is one shipped line. Instance and Source are always set so
// a merged result can be read without a second lookup.
type AggregatedLogEntry struct {
	Instance  string `json:"instance"`
	Source    string `json:"source"`
	Timestamp string `json:"timestamp,omitempty"`
	Line      string `json:"line"`
}

// AggregatedLogResult is the response, including which instances contributed so
// "no match" can be read against "these instances were actually searched".
type AggregatedLogResult struct {
	Entries []AggregatedLogEntry `json:"items"`
	// Instances lists every instance that has shipped at least one line, newest
	// shipper first. It is what makes an empty result interpretable.
	Instances []string `json:"instances"`
	// Truncated reports that older matches exist beyond Limit.
	Truncated bool `json:"truncated"`
	// Match echoes the full-text expression that was applied, so a caller can
	// tell "no results" from "the mode I asked for was not used".
	Match string `json:"match,omitempty"`
	// Stored and Dropped are this process's shipping counters, so a search can
	// tell "that line was never shipped" from "shipping dropped lines".
	Stored  int64 `json:"stored_lines"`
	Dropped int64 `json:"dropped_lines"`
}

// AggregatedLogQueryService searches the lines instances shipped into the
// shared database. It answers the recorded gap "multi-instance deployments can
// only see their own files": with shipping on, one search covers every instance
// that is writing to the same database.
type AggregatedLogQueryService struct {
	db *gorm.DB
	// sink is optional: it only supplies this process's stored/dropped counters.
	sink *InstanceLogSink
}

func NewAggregatedLogQueryService(db *gorm.DB) *AggregatedLogQueryService {
	return &AggregatedLogQueryService{db: db}
}

// WithSink attaches this process's shipping counters to the response.
func (s *AggregatedLogQueryService) WithSink(sink *InstanceLogSink) *AggregatedLogQueryService {
	s.sink = sink
	return s
}

// Query returns the newest matching lines across every instance.
func (s *AggregatedLogQueryService) Query(ctx context.Context, q AggregatedLogQuery) (AggregatedLogResult, error) {
	result := AggregatedLogResult{Entries: []AggregatedLogEntry{}}
	if s == nil || s.db == nil {
		return result, nil
	}
	if s.sink != nil {
		result.Stored = s.sink.Stored()
		result.Dropped = s.sink.Dropped()
	}
	limit := q.Limit
	if limit <= 0 {
		limit = DefaultLogQueryLimit
	}
	if limit > MaxLogQueryLimit {
		limit = MaxLogQueryLimit
	}

	instances, err := s.instances(ctx)
	if err != nil {
		return result, err
	}
	result.Instances = instances

	query := s.db.WithContext(ctx).Model(&models.InstanceLogEntry{})
	if trimmed := strings.TrimSpace(q.Query); trimmed != "" {
		query = query.Where("line ILIKE ?", "%"+escapeLikePattern(trimmed)+"%")
	}
	if trimmed := strings.TrimSpace(q.Match); trimmed != "" {
		result.Match = trimmed
		// websearch_to_tsquery is the forgiving entry point: it accepts what an
		// operator types (words, "a phrase", OR, -excluded) and never raises a
		// syntax error, unlike to_tsquery. The 'simple' configuration is
		// deliberate - log lines are identifiers and stack frames, and language
		// stemming would rewrite `connection` into `connect` and match lines the
		// caller did not ask for.
		query = query.Where("line_tsv @@ websearch_to_tsquery('simple', ?)", trimmed)
	}
	if trimmed := strings.TrimSpace(q.Component); trimmed != "" {
		// Same whole-token rule as the file search: component=api must not match
		// component=api_auth.
		// The parentheses are load-bearing: a raw condition is joined with AND,
		// and without them the OR would escape the rest of the filter.
		query = query.Where("(line ILIKE ? OR line ILIKE ?)",
			"%component="+escapeLikePattern(trimmed)+" %",
			"%component="+escapeLikePattern(trimmed))
	}
	if q.Since != nil {
		query = query.Where("logged_at >= ?", q.Since.UTC())
	}
	if q.Until != nil {
		query = query.Where("logged_at <= ?", q.Until.UTC())
	}
	if trimmed := strings.TrimSpace(q.Instance); trimmed != "" {
		query = query.Where("instance_id = ?", trimmed)
	}
	if trimmed := strings.TrimSpace(q.Source); trimmed != "" {
		query = query.Where("source = ?", trimmed)
	}

	// One extra row is the cheapest way to answer "are there older matches?"
	// without a second count query over the whole table.
	var rows []models.InstanceLogEntry
	if err := query.
		Order("logged_at DESC, id DESC").
		Limit(limit + 1).
		Find(&rows).Error; err != nil {
		return result, err
	}
	if len(rows) > limit {
		result.Truncated = true
		rows = rows[:limit]
	}
	for _, row := range rows {
		result.Entries = append(result.Entries, AggregatedLogEntry{
			Instance:  row.InstanceID,
			Source:    row.Source,
			Timestamp: row.LoggedAt.UTC().Format(time.RFC3339Nano),
			Line:      row.Line,
		})
	}
	return result, nil
}

// instances lists the instances that have shipped lines, so an empty search
// result still says whether anyone is shipping at all.
func (s *AggregatedLogQueryService) instances(ctx context.Context) ([]string, error) {
	var ids []string
	err := s.db.WithContext(ctx).
		Model(&models.InstanceLogEntry{}).
		Distinct().
		Order("instance_id").
		Pluck("instance_id", &ids).Error
	if err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []string{}
	}
	return ids, nil
}

// escapeLikePattern neutralises the LIKE wildcards in a caller-supplied string,
// so a search for "100%" looks for a literal percent rather than matching
// everything. The file scanner does a plain substring match, so the aggregated
// path has to agree with it.
func escapeLikePattern(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}
