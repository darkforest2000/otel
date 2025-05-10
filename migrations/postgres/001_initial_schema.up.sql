CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    surname VARCHAR(255) NOT NULL
);

INSERT INTO users (name, surname) VALUES ('John', 'Doe');
INSERT INTO users (name, surname) VALUES ('Jane', 'Smith');
INSERT INTO users (name, surname) VALUES ('Jim', 'Beam');