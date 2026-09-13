# Course-runner outer seccomp profile

`bubblewrap-outer.json` is derived from Moby profiles release `seccomp/v0.2.3`,
commit `836ae4d37ef2ec995c77c99fc55f5b5f3af3a897`. The unmodified upstream
`seccomp/default.json` SHA-256 is
`536529b665dd0972c37bfb569f5d4ac8a53592e7b00752bc39ff063ca9864c74`.
The reviewed derived `bubblewrap-outer.json` SHA-256 is
`f6ed706bb4af6e5b602f12d86ceaf254a365ab2f0f7cabf0a9f56243044683fa`.

The only policy addition allows `clone`, `unshare`, `mount`, `umount2`, and
`pivot_root` so the non-root Bubblewrap process can construct a nested user,
mount, PID, IPC, UTS, cgroup, and network namespace. The container still drops
all capabilities, keeps `no-new-privileges`, uses a read-only root filesystem,
and retains every other rule from Moby's default allowlist.

Before any generated or learner-owned process starts, course-runner passes a
second classic BPF filter to Bubblewrap. That child filter rejects mount and
namespace syscalls again, rejects namespace flags to `clone`, and makes
`clone3` fall back to inspectable `clone`. This split lets Bubblewrap set up the
sandbox without leaving the extra syscalls available to sandboxed content.

The upstream Apache-2.0 license is retained in `LICENSE.moby-profiles`.

Compose also mounts this exact file read-only into course-runner. When learner
verification is enabled, the service computes the mounted bytes' SHA-256 at
startup and requires the v2 contract digest before it runs the fixed Go
preflight. That preflight checks non-root identity, read-only source, hidden
host application paths, unavailable external networking, exact rlimits, and
rejection of a nested user namespace before capability becomes available.
