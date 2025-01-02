-- +goose Up
CREATE TABLE chirps(
	id UUID, 
	user_id UUID NOT NULL, 
	created_at TIMESTAMP NOT NULL, 
	updated_at TIMESTAMP NOT NULL, 
	body VARCHAR(140) NOT NULL, -- SQLC reads this so make sure its NOT NULL

	PRIMARY KEY(id),
	FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE chirps;


