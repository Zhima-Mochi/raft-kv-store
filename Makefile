.PHONY: proto

proto:
	protoc --go_out=. --go-grpc_out=. \
	       --go_opt=module=github.com/Zhima-Mochi/raft-kv-store \
	       --go-grpc_opt=module=github.com/Zhima-Mochi/raft-kv-store \
	       -I proto proto/*.proto