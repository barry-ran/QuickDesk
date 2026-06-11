// Launcher for the legacy remote.html session.
//
// §2.18 / §2.6: the access_code MUST NOT appear in the remote.html URL.
// The Vue shell already performed POST /v1/devices/:id/access-code:verify
// and obtained a one-shot signal_token (60s TTL). We hand off to the
// remote window via two channels:
//
//   • URL:            ?server=&device=&st=<signal_token>&codec=
//     Only the one-shot token goes on the wire.
//   • sessionStorage: quickdesk_remote_handoff__<signal_token> = {access_code}
//     The SPAKE2 auth still needs the access_code as shared secret
//     (§2.6 "同样可验证"). sessionStorage is same-origin only and is
//     cleared by remote-main.js after it reads the entry.

export function openRemoteSession({ deviceId, signalToken, accessCode }) {
  const serverUrl = localStorage.getItem('quickdesk_signaling_url')
    || 'ws://qdsignaling.quickcoder.cc:8000'
  const codec = localStorage.getItem('quickdesk_video_codec') || 'AV1'

  // Stash the access_code keyed by signal_token so the remote window can
  // pick it up. sessionStorage is shared with same-origin tabs opened
  // via window.open(same-origin target).
  try {
    const key = `quickdesk_remote_handoff__${signalToken}`
    sessionStorage.setItem(key, JSON.stringify({
      access_code: accessCode,
      device_id: deviceId,
      created_at: Date.now(),
    }))
  } catch { /* storage full / disabled — SPAKE2 will fail loudly */ }

  const params = new URLSearchParams({
    server: serverUrl,
    device: deviceId,
    st:     signalToken,
    codec,
  })

  const url = `remote.html?${params.toString()}`
  const isMobile = /Android|iPhone|iPad|iPod/i.test(navigator.userAgent)
  if (isMobile) {
    window.location.href = url
  } else {
    window.open(url, `quickdesk_${deviceId}`)
  }
}
