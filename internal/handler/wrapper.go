package handler

type envelope[T any] struct {
	Data T `json:"data"`
}
