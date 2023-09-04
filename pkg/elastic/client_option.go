package elastic

type ClientOptions struct {
	Url           string
	BasicAuthUser string
	BasicAuthPass string
	Headers       []string
}
