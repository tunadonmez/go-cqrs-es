module github.com/techbank/account-common

go 1.23

require github.com/techbank/cqrs-core v0.0.0

require go.mongodb.org/mongo-driver v1.17.1 // indirect

replace github.com/techbank/cqrs-core => ../cqrs-core
