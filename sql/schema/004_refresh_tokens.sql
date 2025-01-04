-- +goose Up
CREATE TABLE refresh_tokens (
	token VARCHAR(64), -- 32 byte hex encoded string. each byte becomes 2 hex characters
	created_at TIMESTAMP NOT NULL, 
	updated_at TIMESTAMP NOT NULL, 
	user_id UUID NOT NULL, 
	expires_at TIMESTAMP NOT NULL, 
	revoked_at TIMESTAMP, 

	PRIMARY KEY(token),
	FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE refresh_tokens;
