package types

import "net/http"

// LogUpdateRequest - update log req
type LogUpdateRequest struct {
	ID string `json:"-"`
	// BucketID        string        `json:"-"`
	StatusCode      int           `json:"status_code"`
	ResponseBody    []byte        `json:"response_body"`
	ResponseHeaders http.Header   `json:"response_headers"`
	Status          RequestStatus `json:"status"`
	Retries         int           `json:"retries"`
}

// RequestStatus - request status
type RequestStatus int

// default statuses
const (
	RequestStatusPreparing RequestStatus = iota
	RequestStatusSent
	RequestStatusFailed
	RequestStatusStalled // if request destination wasn't listening - incoming requests will be stalled
	RequestStatusReceived
	RequestStatusRejected
)

func (s RequestStatus) String() string {
	switch s {
	case RequestStatusPreparing:
		return "preparing"
	case RequestStatusSent:
		return "sent"
	case RequestStatusFailed:
		return "failed"
	case RequestStatusStalled:
		return "stalled"
	case RequestStatusReceived:
		return "received"
	case RequestStatusRejected:
		return "rejected"
	default:
		return "unknown"
	}
}

// RequestStatusFromCode - gets request status from resp status code
func RequestStatusFromCode(code int) RequestStatus {
	if code >= 200 && code <= 300 {
		return RequestStatusSent
	}

	return RequestStatusFailed
}
