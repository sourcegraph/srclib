# Sourcegraph binary release process

1. `go install github.com/laxer/goxc`
1. Ensure you have the AWS credentials set so that the AWS CLI (`aws`) can write to the `srclib-release` S3 bucket.
1. Run `make release V=1.2.3`, where `1.2.3` is the version you want to release (which can be arbitrarily chosen but should be the next sequential git release tag for official releases).
