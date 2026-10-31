# backupWithRclone 

Frankly, this project is overkill. Most of what I do here can be achieved
in a different way by multiple dedicated rclone calls. And that's probably what
I would advise you to do.

Originally, I had certain improvements in mind that in the end did not really
make sense and are removed from the project. The rest is probably not sufficiently
helpful for anyone but myself.

However, for me, playing around with different styles of automation was and still is fun.

If you are curious feel free to look into this code.

# Functionality

I'd like to backup basically all of my home directory – except for some files.
This, I want to be a whitelist rather than a blacklist. I want to be able to
backup several files, and not all of them in the same manner. E.g., I want
one directory to store a selected list of movies I want to preserve. This
directory contains soft links to those movies. For backing up this subdir, I
want to use the rsync argument `--copy-links` that I don't want to use for
other directories. 

Originally, there were also other thinks I wanted to change, which turned out 
to be nonsense.

You can configure multiple directories that are supposed to be synced using
`rclone sync` (see `example-config.yaml`).

The configuration should be stored in a directory `~/.backupWithRclone` and named `config.yaml`. You can specify
a different directory with the environment variable `BACKUP_WITH_RCLONE_DIR`.

Concrete config settings and flags should be explained in `example-config.yaml` and by running `backupWithRclone -h`.