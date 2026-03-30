module github.com/techbank/account-common

go 1.26.1

require github.com/techbank/cqrs-core v0.0.0

require go.mongodb.org/mongo-driver/v2 v2.5.0 // indirect

replace github.com/techbank/cqrs-core => ../cqrs-core
