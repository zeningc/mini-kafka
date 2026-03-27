package model

type Message struct {
	Value  string `json:"value"`
	Offset int64  `json:"offset"`
}
