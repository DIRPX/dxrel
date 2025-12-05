package conventional

type Message struct {
	Type     Type
	Scope    Scope
	Subject  Subject
	Body     Body
	Trailers []Trailer
}
