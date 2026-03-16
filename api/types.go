package api

import "github.com/zeningc/mini-kafka/broker"

type ProduceRequest struct	{
	Value string `json:"value"`
}

type ProduceResponse struct	{
	Offset int64 `json:"offset"`
}

type ConsumeResponse struct	{
	Messages []broker.Message `json:"messages"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}