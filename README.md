# backupWithRclone 

Consider this code highly experimental.

The code is not yet in a releasable state. Basic functionality and useful documentation is lacking.
The program structure is questionable. It rather reflects my personal leaning process than
stable software.

That being said: feel free to look into it.

# Functionality
My main issue with `rclone` is that I like to regularly sync directories with many files that
mostly don't change. If I call `rclone sync` on such a directory, it takes ages to discover there
is nothing to do. When only few files changed in some small subdirectory, it still browses the entire tree.

Instead, backupWithRclone calls `rclone sync` only for those parts of a directory tree that have actually
been changed. Also, it can do this for multiple directories.

You can configure multiple directories that are supposed to be synced using `rclone sync` (see `example-config.yaml`). 
`backupWithRclone` searches these directories for recent changes and will perform updates accordingly.

The configuration should be stored in a directory `~/.backupWithRclone` and named `config.yaml`. You can specify
a different directory with the environment variable `BACKUP_WITH_RCLONE_DIR`.

Concrete config settings and flags should be explained in `example-config.yaml` and by running `backupWithRclone -h`.