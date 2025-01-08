# Chirpy: A Twitter Clone

**Chirpy** is a Twitter-inspired guided project from [boot.dev](https://boot.dev). 

---

## Features

- **User Management**: Register, login, and update user information with JWT-based authentication and refresh tokens.
- **Chirp Management**: Create, retrieve, update, and delete chirps 
- **Admin Metrics**: View server statistics and manage application state during development.
- **Webhooks**: Has a mock route in order to demonstrate web hooks
- **Static File Server**: Serve static files under the `/app/` path.
- **Health Check**: Monitor the API status via a health check endpoint.
- To test the api, you can use something like [postman](https://www.postman.com/) or use the index page at localhost:8080/app

---

## Tech Stack

- **Backend**: Go 
- **Database**: PostgreSQL
  - SQL queries are generated using **[sqlc](https://github.com/kyleconroy/sqlc)**.
  - Database migrations are handled with **[goose](https://github.com/pressly/goose)**.
- **Authentication**: JWT tokens with support for refresh tokens.
- **Environment Variables**: Configuration managed via `.env` file and **[godotenv](https://github.com/joho/godotenv)**.

---

## API Endpoints

### Health Check
- `GET /api/healthz`: Returns the health status of the server.

### Metrics and Admin
- `GET /admin/metrics`: Displays server metrics, including file server hits.
- `POST /admin/reset`: Resets server state (enabled only in development).

### Chirps
- `GET /api/chirps`: Retrieves all chirps.
- `GET /api/chirps/{chirpID}`: Retrieves a specific chirp by ID.
- `POST /api/chirps`: Creates a new chirp.
- `DELETE /api/chirps/{chirpID}`: Deletes a chirp by ID.

### Users
- `POST /api/users`: Registers a new user.
- `PUT /api/users`: Updates an existing user's details.

### Authentication
- `POST /api/login`: Logs in a user and provides access/refresh tokens.
- `POST /api/refresh`: Refreshes the user's access token using a valid refresh token.
- `POST /api/revoke`: Revokes a user's refresh token.
