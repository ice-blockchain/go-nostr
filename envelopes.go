package nostr

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

	"github.com/mailru/easyjson"
	jwriter "github.com/mailru/easyjson/jwriter"
	"github.com/tidwall/gjson"
)

var (
	ErrMessageUnknown = errors.New("unknown message")
	ErrMessageParse   = errors.New("parse message")
)

func ParseMessage(message []byte) (Envelope, error) {
	firstComma := bytes.Index(message, []byte{','})
	if firstComma == -1 {
		return nil, ErrMessageUnknown
	}
	label := message[0:firstComma]

	var v Envelope
	switch {
	case bytes.Contains(label, []byte("EVENT")):
		v = &EventEnvelope{}
	case bytes.Contains(label, []byte("REQ")):
		v = &ReqEnvelope{}
	case bytes.Contains(label, []byte("COUNT")):
		v = &CountEnvelope{}
	case bytes.Contains(label, []byte("NOTICE")):
		x := NoticeEnvelope("")
		v = &x
	case bytes.Contains(label, []byte("EOSE")):
		x := EOSEEnvelope("")
		v = &x
	case bytes.Contains(label, []byte("OK")):
		v = &OKEnvelope{}
	case bytes.Contains(label, []byte("AUTH")):
		v = &AuthEnvelope{}
	case bytes.Contains(label, []byte("CLOSED")):
		v = &ClosedEnvelope{}
	case bytes.Contains(label, []byte("CLOSE")):
		x := CloseEnvelope("")
		v = &x
	default:
		return nil, ErrMessageUnknown
	}

	if err := v.UnmarshalJSON(message); err != nil {
		return nil, errors.Join(ErrMessageParse, err)
	}
	return v, nil
}

type Envelope interface {
	Label() string
	UnmarshalJSON([]byte) error
	MarshalJSON() ([]byte, error)
	String() string
}

var (
	_ Envelope = (*EventEnvelope)(nil)
	_ Envelope = (*ReqEnvelope)(nil)
	_ Envelope = (*CountEnvelope)(nil)
	_ Envelope = (*NoticeEnvelope)(nil)
	_ Envelope = (*EOSEEnvelope)(nil)
	_ Envelope = (*CloseEnvelope)(nil)
	_ Envelope = (*OKEnvelope)(nil)
	_ Envelope = (*AuthEnvelope)(nil)
)

type EventEnvelope struct {
	SubscriptionID *string
	Events         []*Event
}

func (_ EventEnvelope) Label() string { return "EVENT" }

func (v EventEnvelope) String() string {
	j, _ := json.Marshal(v)
	return string(j)
}

func (v *EventEnvelope) UnmarshalJSON(data []byte) error {
	r := gjson.ParseBytes(data)
	arr := r.Array()
	switch len(arr) {
	case 0, 1:
		return fmt.Errorf("failed to decode EVENT envelope: unknown array len: %v", len(arr))

	// No subscription ID: ["EVENT", event].
	case 2:
		var ev Event

		err := easyjson.Unmarshal([]byte(arr[1].Raw), &ev)
		if err == nil {
			v.Events = []*Event{&ev}

			return nil
		}

		return err

	// With multiple events: ["EVENT", [optional subscriptionID], <event1>, [event2], ...].
	default:
		jsonEvents := arr[1:] // Skip the first element, which is the label.
		if jsonEvents[0].Type == gjson.String {
			v.SubscriptionID = &jsonEvents[0].Str
			jsonEvents = jsonEvents[1:]
		} else if jsonEvents[0].Type == gjson.Null {
			v.SubscriptionID = nil
			jsonEvents = jsonEvents[1:]
		}
		v.Events = make([]*Event, 0, len(jsonEvents))
		for i := range jsonEvents {
			var ev Event
			if err := easyjson.Unmarshal([]byte(jsonEvents[i].Raw), &ev); err != nil {
				return fmt.Errorf("%w -- on event %d", err, i)
			}
			v.Events = append(v.Events, &ev)
		}
	}
	return nil
}

func (v EventEnvelope) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{NoEscapeHTML: true}
	w.RawString(`["EVENT",`)
	if v.SubscriptionID != nil {
		w.RawString(`"` + *v.SubscriptionID + `"`)
		if len(v.Events) > 0 {
			w.RawByte(',')
		}
	}

	for i := range v.Events {
		v.Events[i].MarshalEasyJSON(&w)
		if i < len(v.Events)-1 {
			w.RawByte(',')
		}
	}
	w.RawString(`]`)

	return w.BuildBytes()
}

type ReqEnvelope struct {
	SubscriptionID string
	Filters
}

func (_ ReqEnvelope) Label() string { return "REQ" }

func (v *ReqEnvelope) UnmarshalJSON(data []byte) error {
	r := gjson.ParseBytes(data)
	arr := r.Array()
	if len(arr) < 3 {
		return fmt.Errorf("failed to decode REQ envelope: missing filters")
	}
	v.SubscriptionID = arr[1].Str
	v.Filters = make(Filters, len(arr)-2)
	f := 0
	for i := 2; i < len(arr); i++ {
		if err := easyjson.Unmarshal([]byte(arr[i].Raw), &v.Filters[f]); err != nil {
			return fmt.Errorf("%w -- on filter %d", err, f)
		}
		f++
	}

	return nil
}

func (v ReqEnvelope) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{NoEscapeHTML: true}
	w.RawString(`["REQ",`)
	w.RawString(`"` + v.SubscriptionID + `"`)
	for _, filter := range v.Filters {
		w.RawString(`,`)
		filter.MarshalEasyJSON(&w)
	}
	w.RawString(`]`)
	return w.BuildBytes()
}

type CountEnvelope struct {
	SubscriptionID string
	Filters
	Count       *int64
	HyperLogLog []byte
}

func (_ CountEnvelope) Label() string { return "COUNT" }
func (c CountEnvelope) String() string {
	v, _ := json.Marshal(c)
	return string(v)
}

func (v *CountEnvelope) UnmarshalJSON(data []byte) error {
	r := gjson.ParseBytes(data)
	arr := r.Array()
	if len(arr) < 3 {
		return fmt.Errorf("failed to decode COUNT envelope: missing filters")
	}
	v.SubscriptionID = arr[1].Str

	var countResult struct {
		Count *int64 `json:"count"`
		HLL   string `json:"hll"`
	}
	if err := json.Unmarshal([]byte(arr[2].Raw), &countResult); err == nil && countResult.Count != nil {
		v.Count = countResult.Count
		if len(countResult.HLL) == 512 {
			v.HyperLogLog, err = hex.DecodeString(countResult.HLL)
			if err != nil {
				return fmt.Errorf("invalid \"hll\" value in COUNT message: %w", err)
			}
		}
		return nil
	}

	v.Filters = make(Filters, len(arr)-2)
	f := 0
	for i := 2; i < len(arr); i++ {
		item := []byte(arr[i].Raw)

		if err := easyjson.Unmarshal(item, &v.Filters[f]); err != nil {
			return fmt.Errorf("%w -- on filter %d", err, f)
		}

		f++
	}

	return nil
}

func (v CountEnvelope) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{NoEscapeHTML: true}
	w.RawString(`["COUNT",`)
	w.RawString(`"` + v.SubscriptionID + `"`)
	if v.Count != nil {
		w.RawString(`,{"count":`)
		w.RawString(strconv.FormatInt(*v.Count, 10))
		if v.HyperLogLog != nil {
			w.RawString(`,"hll":"`)
			hllHex := make([]byte, 512)
			hex.Encode(hllHex, v.HyperLogLog)
			w.Buffer.AppendBytes(hllHex)
			w.RawString(`"`)
		}
		w.RawString(`}`)
	} else {
		for _, filter := range v.Filters {
			w.RawString(`,`)
			filter.MarshalEasyJSON(&w)
		}
	}
	w.RawString(`]`)
	return w.BuildBytes()
}

type NoticeEnvelope string

func (_ NoticeEnvelope) Label() string { return "NOTICE" }
func (n NoticeEnvelope) String() string {
	v, _ := json.Marshal(n)
	return string(v)
}

func (v *NoticeEnvelope) UnmarshalJSON(data []byte) error {
	r := gjson.ParseBytes(data)
	arr := r.Array()
	if len(arr) < 2 {
		return fmt.Errorf("failed to decode NOTICE envelope")
	}
	*v = NoticeEnvelope(arr[1].Str)
	return nil
}

func (v NoticeEnvelope) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{NoEscapeHTML: true}
	w.RawString(`["NOTICE",`)
	w.Raw(json.Marshal(string(v)))
	w.RawString(`]`)
	return w.BuildBytes()
}

type EOSEEnvelope string

func (_ EOSEEnvelope) Label() string { return "EOSE" }
func (e EOSEEnvelope) String() string {
	v, _ := json.Marshal(e)
	return string(v)
}

func (v *EOSEEnvelope) UnmarshalJSON(data []byte) error {
	r := gjson.ParseBytes(data)
	arr := r.Array()
	if len(arr) < 2 {
		return fmt.Errorf("failed to decode EOSE envelope")
	}
	*v = EOSEEnvelope(arr[1].Str)
	return nil
}

func (v EOSEEnvelope) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{NoEscapeHTML: true}
	w.RawString(`["EOSE",`)
	w.Raw(json.Marshal(string(v)))
	w.RawString(`]`)
	return w.BuildBytes()
}

type CloseEnvelope string

func (_ CloseEnvelope) Label() string { return "CLOSE" }
func (c CloseEnvelope) String() string {
	v, _ := json.Marshal(c)
	return string(v)
}

func (v *CloseEnvelope) UnmarshalJSON(data []byte) error {
	r := gjson.ParseBytes(data)
	arr := r.Array()
	switch len(arr) {
	case 2:
		*v = CloseEnvelope(arr[1].Str)
		return nil
	default:
		return fmt.Errorf("failed to decode CLOSE envelope")
	}
}

func (v CloseEnvelope) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{NoEscapeHTML: true}
	w.RawString(`["CLOSE",`)
	w.Raw(json.Marshal(string(v)))
	w.RawString(`]`)
	return w.BuildBytes()
}

type ClosedEnvelope struct {
	SubscriptionID string
	Reason         string
}

func (_ ClosedEnvelope) Label() string { return "CLOSED" }
func (c ClosedEnvelope) String() string {
	v, _ := json.Marshal(c)
	return string(v)
}

func (v *ClosedEnvelope) UnmarshalJSON(data []byte) error {
	r := gjson.ParseBytes(data)
	arr := r.Array()
	switch len(arr) {
	case 3:
		*v = ClosedEnvelope{arr[1].Str, arr[2].Str}
		return nil
	default:
		return fmt.Errorf("failed to decode CLOSED envelope")
	}
}

func (v ClosedEnvelope) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{NoEscapeHTML: true}
	w.RawString(`["CLOSED",`)
	w.Raw(json.Marshal(string(v.SubscriptionID)))
	w.RawString(`,`)
	w.Raw(json.Marshal(v.Reason))
	w.RawString(`]`)
	return w.BuildBytes()
}

type OKEnvelope struct {
	EventID string
	OK      bool
	Reason  string
}

func (_ OKEnvelope) Label() string { return "OK" }
func (o OKEnvelope) String() string {
	v, _ := json.Marshal(o)
	return string(v)
}

func (v *OKEnvelope) UnmarshalJSON(data []byte) error {
	r := gjson.ParseBytes(data)
	arr := r.Array()
	if len(arr) < 4 {
		return fmt.Errorf("failed to decode OK envelope: missing fields")
	}
	v.EventID = arr[1].Str
	v.OK = arr[2].Raw == "true"
	v.Reason = arr[3].Str

	return nil
}

func (v OKEnvelope) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{NoEscapeHTML: true}
	w.RawString(`["OK",`)
	w.RawString(`"` + v.EventID + `",`)
	ok := "false"
	if v.OK {
		ok = "true"
	}
	w.RawString(ok)
	w.RawString(`,`)
	w.Raw(json.Marshal(v.Reason))
	w.RawString(`]`)
	return w.BuildBytes()
}

type AuthEnvelope struct {
	Challenge *string
	Event     Event
}

func (_ AuthEnvelope) Label() string { return "AUTH" }
func (a AuthEnvelope) String() string {
	v, _ := json.Marshal(a)
	return string(v)
}

func (v *AuthEnvelope) UnmarshalJSON(data []byte) error {
	r := gjson.ParseBytes(data)
	arr := r.Array()
	if len(arr) < 2 {
		return fmt.Errorf("failed to decode Auth envelope: missing fields")
	}
	if arr[1].IsObject() {
		return easyjson.Unmarshal([]byte(arr[1].Raw), &v.Event)
	} else {
		v.Challenge = &arr[1].Str
	}
	return nil
}

func (v AuthEnvelope) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{NoEscapeHTML: true}
	w.RawString(`["AUTH",`)
	if v.Challenge != nil {
		w.Raw(json.Marshal(*v.Challenge))
	} else {
		v.Event.MarshalEasyJSON(&w)
	}
	w.RawString(`]`)
	return w.BuildBytes()
}
