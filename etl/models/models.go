package models

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/insolar/block-explorer/etl/types"
)

type RecordType string

func RecordTypeFromTypes(rt types.RecordType) RecordType {
	return []RecordType{"state", "request", "result"}[rt]
}

const (
	State   RecordType = "state"
	Request RecordType = "request"
	Result  RecordType = "result"
)

type Reference []byte

func ReferenceFromTypes(r types.Reference) Reference {
	return Reference(r)
}

type Record struct {
	Reference           Reference `gorm:"primary_key;auto_increment:false"`
	Type                RecordType
	ObjectReference     Reference
	PrototypeReference  Reference
	Payload             []byte
	PrevRecordReference Reference
	Hash                []byte
	RawData             []byte
	JetID               string
	PulseNumber         int64
	Order               int
	Timestamp           int64
}

type JetDrop struct {
	PulseNumber    int64  `gorm:"primary_key;auto_increment:false"`
	JetID          string `gorm:"primary_key;auto_increment:false;default:''"`
	FirstPrevHash  []byte
	SecondPrevHash []byte
	Hash           []byte
	RawData        []byte
	Timestamp      int64
	RecordAmount   int
}

func (j *JetDrop) Siblings() []string {
	siblings := []string{j.JetID, fmt.Sprintf("%s0", j.JetID), fmt.Sprintf("%s1", j.JetID)}
	sz := len(j.JetID)
	if sz > 0 {
		siblings = append(siblings, j.JetID[:sz-1])
	}
	return siblings
}

type Pulse struct {
	PulseNumber     int64 `gorm:"primary_key;auto_increment:false"`
	PrevPulseNumber int64
	NextPulseNumber int64
	IsComplete      bool
	IsSequential    bool
	Timestamp       int64
	JetDropAmount   int64
	RecordAmount    int64
}

type JetDropID struct {
	JetID       string
	PulseNumber int64
}

func NewJetDropID(jetID string, pulseNumber int64) *JetDropID {
	tmp := jetID
	if jetID == "*" {
		tmp = ""
	}
	return &JetDropID{JetID: tmp, PulseNumber: pulseNumber}
}

// jetIDRegexp uses for a validation of the JetID
var jetIDRegexp = regexp.MustCompile(`^(\*|([0-1]{1,216}))$`)

// NewJetDropIDFromString create JetDropID from provided string representation. Jet with empty prefix returned with empty jetID.
func NewJetDropIDFromString(jetDropID string) (*JetDropID, error) {
	var pulse int64
	jetDropID, err := url.QueryUnescape(jetDropID)
	if err != nil {
		return nil, fmt.Errorf("wrong jet drop id format")
	}
	s := strings.Split(jetDropID, ":")
	if len(s) != 2 {
		return nil, fmt.Errorf("wrong jet drop id format")
	}
	if !jetIDRegexp.MatchString(s[0]) {
		return nil, fmt.Errorf("wrong jet drop id format")
	}
	pulse, err = strconv.ParseInt(s[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("wrong jet drop id format")
	}

	return NewJetDropID(s[0], pulse), nil
}

func (j *JetDropID) ToString() string {
	return fmt.Sprintf("%s:%d", j.JetIDToString(), j.PulseNumber)
}

func (j *JetDropID) JetIDToString() string {
	tmp := j.JetID
	if j.JetID == "" {
		tmp = "*"
	}
	return tmp
}
