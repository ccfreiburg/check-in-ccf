-- Seed data for event_stats: 8 past events showing attendance trends per group.
-- Dates are Sundays in the 9:30 service running back from April 2026.
-- Groups match the actual synced_group_memberships in checkin.db.

INSERT OR IGNORE INTO event_stats (event_date, group_id, group_name, registered, checked_in, checked_out) VALUES
-- 2026-02-22
('2026-02-22', 599, 'KinderKirche Kinder (9.30)',  18, 14, 13),
('2026-02-22', 602, 'Lion''s Babies Kinder (9.30)', 12,  9,  8),
('2026-02-22', 605, 'Lion''s Kids Kinder (9.30)',   16, 13, 12),
('2026-02-22', 721, 'New Generation Kids (9.30)',    6,  4,  4),
-- 2026-03-01
('2026-03-01', 599, 'KinderKirche Kinder (9.30)',  21, 17, 16),
('2026-03-01', 602, 'Lion''s Babies Kinder (9.30)', 14, 11, 10),
('2026-03-01', 605, 'Lion''s Kids Kinder (9.30)',   19, 15, 14),
('2026-03-01', 721, 'New Generation Kids (9.30)',    7,  5,  5),
-- 2026-03-08
('2026-03-08', 599, 'KinderKirche Kinder (9.30)',  24, 20, 19),
('2026-03-08', 602, 'Lion''s Babies Kinder (9.30)', 17, 13, 12),
('2026-03-08', 605, 'Lion''s Kids Kinder (9.30)',   22, 18, 17),
('2026-03-08', 721, 'New Generation Kids (9.30)',    9,  7,  6),
-- 2026-03-15
('2026-03-15', 599, 'KinderKirche Kinder (9.30)',  20, 16, 15),
('2026-03-15', 602, 'Lion''s Babies Kinder (9.30)', 15, 12, 11),
('2026-03-15', 605, 'Lion''s Kids Kinder (9.30)',   18, 14, 13),
('2026-03-15', 721, 'New Generation Kids (9.30)',    8,  6,  6),
-- 2026-03-22
('2026-03-22', 599, 'KinderKirche Kinder (9.30)',  27, 22, 21),
('2026-03-22', 602, 'Lion''s Babies Kinder (9.30)', 20, 16, 15),
('2026-03-22', 605, 'Lion''s Kids Kinder (9.30)',   25, 20, 19),
('2026-03-22', 721, 'New Generation Kids (9.30)',   10,  8,  7),
-- 2026-03-29
('2026-03-29', 599, 'KinderKirche Kinder (9.30)',  25, 21, 20),
('2026-03-29', 602, 'Lion''s Babies Kinder (9.30)', 18, 14, 14),
('2026-03-29', 605, 'Lion''s Kids Kinder (9.30)',   23, 18, 18),
('2026-03-29', 721, 'New Generation Kids (9.30)',    9,  7,  7),
-- 2026-04-05
('2026-04-05', 599, 'KinderKirche Kinder (9.30)',  30, 25, 24),
('2026-04-05', 602, 'Lion''s Babies Kinder (9.30)', 22, 18, 17),
('2026-04-05', 605, 'Lion''s Kids Kinder (9.30)',   28, 23, 22),
('2026-04-05', 721, 'New Generation Kids (9.30)',   12, 10,  9),
-- 2026-04-12
('2026-04-12', 599, 'KinderKirche Kinder (9.30)',  28, 23, 22),
('2026-04-12', 602, 'Lion''s Babies Kinder (9.30)', 21, 17, 16),
('2026-04-12', 605, 'Lion''s Kids Kinder (9.30)',   26, 21, 20),
('2026-04-12', 721, 'New Generation Kids (9.30)',   11,  9,  8);
