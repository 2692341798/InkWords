# Chromium outer seccomp profile

The Moby baseline is `seccomp/v0.2.3`, commit
`836ae4d37ef2ec995c77c99fc55f5b5f3af3a897`, already vendored for course-runner.
Its unmodified `seccomp/default.json` SHA256 is
`536529b665dd0972c37bfb569f5d4ac8a53592e7b00752bc39ff063ca9864c74`.
The upstream Apache-2.0 license is preserved alongside this file.

`chromium-outer.json` preserves every baseline rule and adds only `clone`,
`setns`, `unshare`, and `chroot` for the trusted Chromium sandbox setup.
It does not include course-runner's extra mount, umount2, or pivot_root rules.
SHA256: `a48cc338e4c04001707da2279d26d8ba7e6700c15ccf83735b085af6763c4a5f`.

Use only with the non-root, cap-drop ALL, no-new-privileges, read-only root,
bounded tmpfs/memory/PID/CPU controls in `docker-compose.textbook-pdf.yml`.
No host PID/network/IPC, Docker socket, privileged mode, setuid helper, or
disabled browser sandbox is required. The Chromium renderer establishes its
own user/PID/network namespaces and adds a second seccomp filter before
rendering content. The browser parent still needs the existing service's
trusted I/O; this profile is not a general-purpose arbitrary-code executor.

See `docs/decisions/textbook-pdf-runtime.md` for actual process evidence and
the distinction from the nested Bubblewrap course-runner browser failure.
