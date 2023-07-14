package main

const CreateTableIfNotExists = `

	CREATE TABLE IF NOT EXISTS users
	(
		id INTEGER PRIMARY KEY,
		assigned_to STRING
	);

	CREATE TABLE IF NOT EXISTS messages
	(
		user_chat_message_id INTEGER,
		support_chat_message_id INTEGER,
		user_id INTEGER,

		FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
	);


`

const BySupportChatMessagID = `
	SELECT user_id, user_chat_message_id 
	FROM messages
	WHERE support_chat_message_id = ?;
`

const ByUserChatMessageID = `
	SELECT user_id, support_chat_message_id
	FROM messages
	WHERE user_chat_message_id = ?;
`