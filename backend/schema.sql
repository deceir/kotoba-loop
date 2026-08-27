CREATE TABLE IF NOT EXISTS users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  email VARCHAR(255) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS decks (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  name VARCHAR(120) NOT NULL,
  description VARCHAR(500) NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS words (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  jmdict_seq BIGINT NULL UNIQUE,
  english VARCHAR(255) NOT NULL,
  japanese VARCHAR(255) NOT NULL,
  reading VARCHAR(255) NOT NULL,
  INDEX idx_words_reading (reading)
);
CREATE TABLE IF NOT EXISTS word_decks (
  word_id BIGINT NOT NULL,
  deck_id BIGINT NOT NULL,
  PRIMARY KEY (word_id, deck_id),
  FOREIGN KEY (word_id) REFERENCES words(id) ON DELETE CASCADE,
  FOREIGN KEY (deck_id) REFERENCES decks(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS user_words (
  user_id BIGINT NOT NULL,
  word_id BIGINT NOT NULL,
  due_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  interval_days INT NOT NULL DEFAULT 0,
  ease_factor DECIMAL(4,2) NOT NULL DEFAULT 2.50,
  repetitions INT NOT NULL DEFAULT 0,
  lapses INT NOT NULL DEFAULT 0,
  PRIMARY KEY (user_id, word_id),
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (word_id) REFERENCES words(id) ON DELETE CASCADE
);
INSERT INTO decks (id, name, description) VALUES (1, 'Everyday Japanese', 'Useful words for daily life') ON DUPLICATE KEY UPDATE name=name;
INSERT INTO words (id, jmdict_seq, english, japanese, reading) VALUES
 (1,1605620,'water','水','みず'),(2,1467640,'cat','猫','ねこ'),(3,1421700,'morning','朝','あさ'),
 (4,1540170,'friend','友達','ともだち'),(5,1486650,'delicious','美味しい','おいしい')
ON DUPLICATE KEY UPDATE english=VALUES(english);
INSERT IGNORE INTO word_decks(word_id,deck_id) VALUES (1,1),(2,1),(3,1),(4,1),(5,1);
