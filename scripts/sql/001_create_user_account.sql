CREATE TABLE `user_account` (
  `id` varchar(36) NOT NULL COMMENT 'User UUID',
  `name` varchar(24) NOT NULL DEFAULT '' COMMENT 'User display name',
  `password` varchar(128) NOT NULL DEFAULT '' COMMENT 'Bcrypt password hash',
  `email` varchar(48) NOT NULL DEFAULT '' COMMENT 'Normalized email address',
  `version` int unsigned NOT NULL DEFAULT '1' COMMENT 'Optimistic lock version',
  `created_at` bigint NOT NULL DEFAULT '0' COMMENT 'Creation timestamp in milliseconds',
  `updated_at` bigint NOT NULL DEFAULT '0' COMMENT 'Update timestamp in milliseconds',
  `deleted_at` bigint NOT NULL DEFAULT '0' COMMENT 'Soft deletion timestamp; 0 means active',
  PRIMARY KEY (`id`),
  KEY `idx_name_deleted` (`name`, `deleted_at`)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_unicode_ci
  COMMENT='User Aggregate persistence';
