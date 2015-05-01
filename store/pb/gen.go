package pb

//go:generate protoc -I../../../../../ -I../../../../../github.com/gogo/protobuf/protobuf -I../../../../../sourcegraph.com/sourcegraph/srclib/graph -I../../../../../sourcegraph.com/sourcegraph/srclib/ann -I. --gogo_out=plugins=grpc:. srcstore.proto
//go:generate gen-mocks -w -i=.+(Server|Client|Service)$ -o mock -outpkg mock -name_prefix= -no_pass_args=opts
