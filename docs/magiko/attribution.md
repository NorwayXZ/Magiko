# Attribution and provenance

- **Magpie**: <https://github.com/yetone/magpie>, MIT, original copyright and
  LICENSE retained. Base commit 588f87a126834c2ed1c6348a35c6f031924c1855.
- **LeilaCrak**: local supplied LeilaCrak-1.0.0-mac-arm64.zip was the source
  of two optional skill directories. No Electron runtime, activation database,
  account tokens, private proxy implementation or unrestricted-prompt files
  are shipped in Magiko.
- `protocol-reverse-engineering`: embedded source's MIT LICENSE credits
  Copyright (c) 2024 Seth Hobson. Package metadata from the supplied artifact
  is retained as source data, not asserted authorship by this fork.
- `reverse-engineering-api`: its supplied MIT LICENSE (Copyright (c) 2025 kalil0321) is retained verbatim.
  Source consists of SKILL.md, Python helper scripts, references and templates.
- **Magiko changes**: application identity, request preference module,
  workspace page, optional pack import, tests, icon and build packaging.
  New source follows the repository MIT license. The geometric icon is generated
  by `build/magiko/icon.py`; Pillow is required only to regenerate it.

The skill texts describe their respective tool workflows. They are not
instructions to the Magiko application at startup and are not automatically
installed into any agent.

Whitespace in extracted skill text/source has been normalized; licence contents
are unchanged. Helper Python sources are parsed for syntax, not executed.
