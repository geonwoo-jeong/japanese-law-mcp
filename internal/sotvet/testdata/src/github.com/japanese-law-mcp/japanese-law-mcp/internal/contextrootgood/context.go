package contextrootgood

import "context"

func child(parent context.Context) context.Context {
	return context.WithValue(parent, key{}, "value")
}

type key struct{}
