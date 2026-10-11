#!/usr/bin/env bash
function md { mkdir -p "$1"; }
rm -rf testdirs
md testdirs
(cd testdirs
md unchanged_path
echo "testdirs" > unchanged_path/test.txt
md changed_path_direct
md changed_path_indirect
echo "testdirs" > changed_path_indirect/test.txt
md changed_subpaths/unchanged
echo "testdirs" > changed_subpaths/unchanged/test1.txt
echo "testdirs" > changed_subpaths/unchanged/test2.txt
md changed_subpaths/unchanged_subdir
md changed_subpaths/changed_subdir_direct
md changed_subpaths/changed_subdir_indirect
echo "testdirs" > changed_subpaths/changed_subdir_indirect/test1.txt
md changed_subpaths/with_changed_subdirs/unchanged_subdir
md changed_subpaths/with_changed_subdirs/changed_subdir1
echo "testdirs" > changed_subpaths/with_changed_subdirs/changed_subdir1/test.txt
md changed_subpaths/with_changed_subdirs/changed_subdir2
echo "testdirs" > changed_subpaths/with_changed_subdirs/changed_subdir2/test1.txt
echo "testdirs" > changed_subpaths/with_changed_subdirs/changed_subdir2/test2.txt)

find "./testdirs" \
  -exec /usr/bin/touch -c -m -t 202202221200.00 {} +

(cd testdirs
touch changed_path_direct
touch changed_path_indirect/test.txt
touch changed_subpaths/changed_subdir_direct
touch changed_subpaths/changed_subdir_indirect/test1.txt
touch changed_subpaths/with_changed_subdirs/changed_subdir1/test.txt
touch changed_subpaths/with_changed_subdirs/changed_subdir2/test2.txt)

# Zu testen sind
# unchanged_path -> []
# changed_path_direct -> ["changed_path_direct"]
# changed_path_indirect -> ["changed_path_indirect"]
# changed_subpaths -> ["changed_subpaths/changed_subdir_direct",
#                      "changed_subpaths/changed_subdir_indirect",
#                      "changed_subpaths/with_changed_subdirs/changed_subdir1",
#                      "changed_subpaths/with_changed_subdirs/changed_subdir2"]
