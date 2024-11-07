CREATE TABLE plans(
  id INTEGER PRIMARY KEY,
  task_id INTEGER,
  activity_id INTEGER,
  location TEXT,
  time_at_after DATETIME, -- in Unix time
  time_before DATETIME, -- in Unix time
  duration_ge INTEGER, -- in seconds
  duration_lt INTEGER, -- in seconds
  FOREIGN KEY(task_id) REFERENCES tasks(id),
  FOREIGN KEY(activity_id) REFERENCES activity_log(id)
);
