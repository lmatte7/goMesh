module github.com/lmatte7/goMesh

go 1.16

replace github.com/lmatte7/go-meshtastic-protobufs => ./go-meshtastic-protobufs

require (
	github.com/jacobsa/go-serial v0.0.0-20180131005756-15cf729a72d4
	github.com/lmatte7/meshtastic-go v0.0.0-20210519183941-63df7c3c97a0
	google.golang.org/protobuf v1.26.0
)
