#!/usr/bin/env bash
source .env
: "${INTEGRATION_TEST_DEST_PREFIX:?INTEGRATION_TEST_DEST_PREFIX is missing or empty – define it in .env}"
export BACKUP_WITH_RCLONE_DIR="./integrationTest/backupDir"
export INTEGRATION_TEST_DEST_PREFIX

integration_test_dest_root="$INTEGRATION_TEST_DEST_PREFIX/integration-test"

# Functions
error() {
  echo ERROR: "$@"
  exit 1
}

file_contains_line() {
    local file="$1"
    local line="$2"

    grep -Fxq -- "$line" "$file"
}

# build
go build || error "Building program failed"

# Reset standard content
integration_test_source_root="./integrationTest/testdirs"
echo "A" > "$integration_test_source_root/standard/dir1/file_a.txt"
echo "B" > "$integration_test_source_root/standard/dir1/file_b.txt"

# mount integration test
echo "Mounting test dir"
test_mount_point="./integrationTest/mnt"
mkdir -p "$test_mount_point"
rclone mount "$INTEGRATION_TEST_DEST_PREFIX" "$test_mount_point" --read-only --daemon \
  || error "rclone mount failed"
trap 'umount "$test_mount_point"' EXIT

# Standard test
echo "Starting standard test"
envsubst < "$BACKUP_WITH_RCLONE_DIR/config-std-tmpl.yaml" > "$BACKUP_WITH_RCLONE_DIR/config.yaml"
rclone purge "$integration_test_dest_root"
./backupWithRclone
ls "$test_mount_point/integration-test"

standard_dir="$test_mount_point/integration-test/standard"
echo "Checking standard dir $standard_dir"
test -f "$standard_dir/dir1/file_a.txt" || error "Backup of standard/dir1/file_a.txt not found."
test -f "$standard_dir/dir1/file_b.txt" || error "Backup of standard/dir1/file_b.txt not found."
test -f "$standard_dir/dir2/file_c.txt" || error "Backup of standard/dir2/file_c.txt not found."
test -f "$standard_dir/dir2/file_d.txt" || error "Backup of standard/dir2/file_d.txt not found."
[[ "$(<$standard_dir/dir1/file_a.txt)" == "A" ]] || error "Content for standard/dir1/file_a.txt is wrong."

exclude_dir="$test_mount_point/integration-test/check_exclude"
echo "Checking exclude dir"
test -f "$exclude_dir/dir/file.txt" || error "Backup of check_exclude/dir/file.txt not found."
test ! -f "$exclude_dir/dir/excluded_file1.txt" || error "Backup of check_exclude/dir/excluded_file1.txt was found."
test ! -f "$exclude_dir/dir/excludesubdir/file.txt" || error "Backup of check_exclude/dir/excludesubdir/file.txt was found."
test ! -f "$exclude_dir/dir/excludesubfile" || error "Backup of check_exclude/dir/excludesubfile was found."
test ! -f "$exclude_dir/excluded_dir1/file.txt" || error "Backup of check_exclude/excluded_dir1/file.txt was found."
test ! -f "$exclude_dir/excluded_dir2/file.txt" || error "Backup of check_exclude/excluded_dir2/file.txt was found."

target_dir="$test_mount_point/integration-test/targetdir"
echo "Checking target dir"
test -f "$target_dir/file.txt" || error "Backup of targetdir/file.txt not found."

linked_files_dir="$test_mount_point/integration-test/check_copy_links"
echo "Checking linked files"
test -f "$linked_files_dir/file_a.txt" || error "Backup of check_copy_links/file_a.txt not found."
test -f "$linked_files_dir/file_b.txt" || error "Backup of check_copy_links/file_b.txt not found."
[[ "$(<$linked_files_dir/file_a.txt)" == "A" ]] || error "Content for check_copy_links/file_a.txt is wrong."

echo "Standard test successful"

echo "Test checking"

# Modify standard content
integration_test_source_root="./integrationTest/testdirs"
echo "X" > "$integration_test_source_root/standard/dir1/file_a.txt"
rm "$integration_test_source_root/standard/dir1/file_b.txt"
rm "$BACKUP_WITH_RCLONE_DIR/size-check"*
rm "$BACKUP_WITH_RCLONE_DIR/checksum-check"*

./backupWithRclone -cs
file_contains_line "$BACKUP_WITH_RCLONE_DIR/size-checks-standard.log" '- dir1/file_b.txt' \
  || error "should report about the missing dir1/file_b.txt"
! file_contains_line "$BACKUP_WITH_RCLONE_DIR/checksum-checks-standard.log" '* dir1/file_b.txt' \
  || error "size check should NOT report about the changed dir1/file_a.txt"

./backupWithRclone -cc
file_contains_line "$BACKUP_WITH_RCLONE_DIR/checksum-checks-standard.log" '- dir1/file_b.txt' \
  || error "should report about the missing dir1/file_b.txt"
file_contains_line "$BACKUP_WITH_RCLONE_DIR/checksum-checks-standard.log" '* dir1/file_a.txt' \
  || error "checksum check should report about the changed dir1/file_a.txt"

echo "Successfully completed checking functionality"
