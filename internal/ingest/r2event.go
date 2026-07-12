package ingest

// r2Event mirrors the body of an R2 event notification message.
// https://developers.cloudflare.com/r2/buckets/event-notifications/
type r2Event struct {
	Account string `json:"account"`
	Action  string `json:"action"`
	Bucket  string `json:"bucket"`
	Object  struct {
		Key  string `json:"key"`
		Size int64  `json:"size"`
	} `json:"object"`
	EventTime string `json:"eventTime"`
}

var createActions = map[string]bool{
	"PutObject":               true,
	"CopyObject":              true,
	"CompleteMultipartUpload": true,
}

func (e r2Event) isCreate() bool {
	return createActions[e.Action]
}
