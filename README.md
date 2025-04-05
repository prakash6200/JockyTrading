# Jocky Trading API

## Overview
Jocky Trading API is a backend service for managing users and transactions in a microservices-based architecture. It is built using GoLang with Fiber and GORM, ensuring high performance and scalability.

## Features
- User authentication and authorization
- API for handling transactions
- PostgreSQL integration using GORM
- Redis caching for improved performance
- Secure JWT-based authentication
- Microservices-ready architecture

## Prerequisites
- Go 1.23 or later
- PostgreSQL
- Redis

## Installation
```sh
git clone https://github.com/prakash6200/JockyTrading.git
cd JockyTrading
go mod tidy
```

## Configuration
Create a `.env` file in the project root and set up your database and other environment variables:
```
DB_HOST=localhost
DB_USER=jocky_user
DB_PASSWORD=jocky_tading
DB_NAME=jocky_db
DB_PORT=5432
JWT_SECRET=your_secret_key
```

## Running the Application
```sh
go run main.go
```

## API Endpoints
### Authentication
- `POST /register` - Register a new user
- `POST /login` - Authenticate user and get a token

### User Management
- `GET /users` - Get all users
- `GET /users/:id` - Get user by ID
- `PUT /users/:id` - Update user details
- `DELETE /users/:id` - Delete a user

### Transactions
- `POST /transactions` - Create a transaction
- `GET /transactions` - Get all transactions
- `GET /transactions/:id` - Get transaction details

### Air for restart server, Development
- `go install github.com/air-verse/air@latest` - Install air

```
export GOPATH=$HOME/xxxxx
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
export PATH=$PATH:$(go env GOPATH)/bin
```

## Contributing
Contributions are welcome! Feel free to submit a pull request or open an issue.

## License
This project is licensed under the MIT License.

## Contact
For any queries, reach out to [prakash6200](https://github.com/prakash6200)  
📧 Email: prakashkumar97068@gmai.com  
📞 Mobile: 6200134797
