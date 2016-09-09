
## Structure of the package

The logwatcher is structured to provide the data-structures and functionality
around _tracking_ and _reading_ log files, but specifically has left out
wrapping this functionality with goroutine specific code.  This lets the user
decide how exactly to incorporate the package.

With regards to the timing of watching files, ultimately `logwatcher` uses
polling.  To avoid missing changes, the polltime should be kept as short as
possible.

## Files changing underneath us.

Logfiles can get rotated and deleted "underneath" the logwatching code.  Some
examples are:

- `Read` is called on a file has been truncated since the last `Read`
  (i.e. data is actually deleted from the file before `logwatcher` has a
  chance to read it).  This is particularly difficult to handle if the new
  (post-truncation) size is larger or equal to the old one.  Assume this
  doesn't happen.

- `Read` is called on a file that has been moved and new data is being written
  to a file with different inode.  We check for this with SameFile().
