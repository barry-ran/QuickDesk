package service

import "testing"

func TestValidateSessionRecordingInput(t *testing.T) {
	tests := []struct {
		name    string
		in      CreateSessionRecordingInput
		wantErr bool
	}{
		{
			name: "valid completed recording",
			in: CreateSessionRecordingInput{
				DeviceID: "123456789",
				Status:   "completed",
			},
		},
		{
			name: "missing device",
			in: CreateSessionRecordingInput{
				Status: "recording",
			},
			wantErr: true,
		},
		{
			name: "invalid status",
			in: CreateSessionRecordingInput{
				DeviceID: "123456789",
				Status:   "paused",
			},
			wantErr: true,
		},
		{
			name: "negative duration",
			in: CreateSessionRecordingInput{
				DeviceID:        "123456789",
				Status:          "completed",
				DurationSeconds: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSessionRecordingInput(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("wantErr=%v got %v", tt.wantErr, err)
			}
		})
	}
}
