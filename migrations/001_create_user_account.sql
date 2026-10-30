-- Intent: store the User Aggregate and serve the User read model.
--   idx_name_deleted_at serves name-prefix lookups; idx_deleted_at_id serves
--   the newest-first keyset page (deleted_at = 0 ORDER BY id DESC, id < ?).
-- Compatibility: creates a new table; no running code depends on it yet.
-- Reversal: DROP TABLE `user_account`; this deletes every stored User.
CREATE TABLE `user_account` (
  `id` varchar(36) NOT NULL COMMENT 'User UUIDv7; sorts in creation order',
  `name` varchar(24) NOT NULL DEFAULT '' COMMENT 'User display name',
  `password` varchar(128) NOT NULL DEFAULT '' COMMENT 'Bcrypt password hash',
  `email` varchar(48) NOT NULL DEFAULT '' COMMENT 'Normalized email address',
  `version` int unsigned NOT NULL DEFAULT '1' COMMENT 'Optimistic lock version',
  `created_at` bigint NOT NULL DEFAULT '0' COMMENT 'Creation timestamp in milliseconds',
  `updated_at` bigint NOT NULL DEFAULT '0' COMMENT 'Update timestamp in milliseconds',
  `deleted_at` bigint NOT NULL DEFAULT '0' COMMENT 'Soft deletion timestamp; 0 means active',
  PRIMARY KEY (`id`),
  KEY `idx_name_deleted_at` (`name`, `deleted_at`),
  KEY `idx_deleted_at_id` (`deleted_at`, `id`)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_unicode_ci
  COMMENT='User Aggregate persistence';
