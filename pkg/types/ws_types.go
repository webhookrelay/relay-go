package types

type EventMeta struct {
	ID                string `json:"id"`
	BucketID          string `json:"bucked_id"`
	BucketName        string `json:"bucket_name"`
	InputID           string `json:"input_id"`
	InputName         string `json:"input_name"`
	OutputName        string `json:"output_name"`
	OutputDestination string `json:"output_destination"`
}

// Event is used by the socket server to stream new webhooks, example:
// {
// 	"type": "webhook",
// 	"meta": {
// 	  "bucked_id": "1593fe5f-45f9-45cc-ba23-675fdc7c1638",
// 	  "bucket_name": "123",
// 	  "input_id": "b90f2fe9-621d-4290-9e74-edd5b61325dd",
// 	  "input_name": "Default public endpoint"
// 	},
// 	"headers": {
// 	  "Accept": [
// 		"*/*"
// 	  ],
// 	  "Content-Length": [
// 		"15"
// 	  ],
// 	  "User-Agent": [
// 		"insomnia/6.2.0"
// 	  ],
// 	  "Cookie": [
// 		"JSESSIONID.9cbae22a=q3lczkhqmp3s15ssahn1hpsta"
// 	  ],
// 	  "Content-Type": [
// 		"application/json"
// 	  ]
// 	},
// 	"query": "foo=bar",
// 	"body": "{\"hi\": \"there\"}",
// 	"method": "PUT"
//   }
type Event struct {
	Type     string              `json:"type"`
	Meta     EventMeta           `json:"meta"`
	Headers  map[string][]string `json:"headers"`
	RawQuery string              `json:"query"`
	Body     string              `json:"body"`
	Method   string              `json:"method"`

	// combined fields from status
	Status  string `json:"status"`
	Message string `json:"message"`
}

// SubscribeRequest contains bin ID
// Example authentication:
// SEND:
// {
//   "action":"auth",
//   "key":"YOUR_KEY"
//   "secret":"YOUR_SECRET"
// }
// Example Subscribe:
// SEND:
// {
//   "action":"subscribe",
//   "buckets": ["bucket-x", "bucket-uuid-here"]
// }
type ActionRequest struct {
	// Action - auth/subscribe
	Action string `json:"action"`

	Key    string `json:"key"`
	Secret string `json:"secret"`

	// if action == subscribe
	Buckets []string `json:"buckets"`
}

type EventStatus struct {
	ID         string `json:"id"`
	StatusCode int    `json:"status_code"`
	Retries    int    `json:"retries"`
	Message    string `json:"message"`
}
