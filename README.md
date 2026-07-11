# tachyne-gw-java-776

> tachyne is an unofficial fan project, not affiliated with Mojang,
> Microsoft, or Minecraft's developer/publisher in any way. See the
> Disclaimer at the bottom.

## Project status

**Work in progress.** tachyne is young and moving fast: a full survival game
runs today, but expect rough edges, missing vanilla features, and breaking
changes between updates. **Bug reports are genuinely useful** — please open a
GitHub Issue with your client version/edition and what you saw. Contributions
are welcome too: see [CONTRIBUTING.md](CONTRIBUTING.md).

**Just want to run a server?** The [quickstart repo](https://github.com/tachyne/tachyne)
brings up the whole stack in one command — Docker Compose or Kubernetes,
classic infinite survival by default, real-Cape-Town earth mode as a variant.

**What's implemented?** Gameplay features live in the world engine, not the
gateways — see [tachyne-world's feature matrix](https://github.com/tachyne/tachyne-world#what-to-expect-vanilla-parity-at-a-glance)
for what to expect (implemented / partial / missing) as a player.




Gateway for Java protocol **776 (Minecraft "26.2")**: terminates real clients,
authorizes logins via **tachyne-access** (fail closed, 30 s cache), attaches
each session to a **tachyne-world** pod over the domain attach protocol, and
renders the typed event stream into wire format. Structurally it is gw-770's
session code **plus a translation boundary**: every clientbound packet is
composed as canonical 770 via `tachyne-common/render770`, then rewritten by
`protocol.TranslatorFor(776)` (the chained 770→…→776 translation from
tachyne-common) at the client edge; serverbound packets are back-translated
776→770 before the shared parsers lift them into typed attach actions.

The 776 config phase is fully version-native: 26.x registries, the complete
`UpdateTagsPacket(776)` tag data, brand + feature flags — all from the shared
`tachyne-common/protocol` composition (generated `tags26x`/`protomap_26x`
data; generators live in `tachyne-world`'s `scripts/`).

Clients arrive via **tachyne-ingress** on `<server-ip>:25565` (handshake
protocol routing + PROXY protocol v1 for real client IPs); this pod's service
is cluster-internal.

## Layout

```
cmd/gw/            entrypoint (env-first config, SIGTERM shutdown)
internal/gw/       gateway: status | login(access) → configuration(26.x) →
                   play (attach session pump ⇄ render770 + TranslatorFor(776))
internal/access/   thin wrapper over tachyne-common/access
internal/wire/     minimal pre-play framing helpers (handshake/status)
deploy/            k8s manifests (StatefulSet + cluster-internal service)
.forgejo/workflows CI: docker build (vet+test inside) + registry push
```

## Configuration (env)

| env                    | meaning                                             |
| ---------------------- | --------------------------------------------------- |
| `TACHYNE_LISTEN`       | client listen address (default `:25565`)            |
| `TACHYNE_BACKEND`      | world attach address (`tachyne-world-0.…:25500`)    |
| `TACHYNE_ATTACH_TOKEN` | attach shared secret (secret `tachyne-attach-token`)|
| `TACHYNE_ACCESS_URL` / `TACHYNE_ACCESS_TOKEN` | tachyne-access (unset = checks off, dev only) |
| `TACHYNE_MOTD`         | server-list description                             |
| `POD_NAME`             | downward API; trailing ordinal = SID                |

## Build / test / deploy

```bash
go build ./... && go test ./...
go run ./cmd/gw
```

CI builds + tags the image on
push to main; `kubectl rollout restart` on the StatefulSet. When a tachyne-common change is involved,
pin the new sha in go.mod and deploy the world pod first. CI module fetches
occasionally time out ("dial <registry>:443 i/o timeout") — an empty-commit
retry works.

## Design notes

- **One gateway build = one client protocol family.** Other versions get
  sibling deployments; ingress routes by handshake protocol. 773–775 are
  currently unserved (deprioritized).
- Adding a future protocol 77x = a new translation step in tachyne-common
  (ID remap + body rewriters for changed packets), a sibling repo copy, and an
  ingress route — the world engine never changes ("worlds are versionless").
- See `tachyne-world/docs/SHARDING.md` for the multi-world-pod plan this
  gateway will implement the client half of (upstream set, rehome).

## Deployment

`Dockerfile` builds a static Go binary into a minimal image. `deploy/` holds
working Kubernetes manifests (the ones this project actually runs) — treat
them as examples: substitute your own image registry, hostnames, namespaces
and secrets before applying them to your cluster.

## Credits

All protocol rendering comes from the shared `tachyne-common` library — see
its credits (PrismarineJS/minecraft-data, misode/mcmeta, the Minecraft Wiki,
ViaVersion as factual references). This repo itself has no third-party
dependencies beyond that library.

## Development transparency

tachyne is built by its maintainer working with an AI coding agent
(Anthropic's Claude): substantial portions of the implementation were written
by the model under human direction, and every change is reviewed, tested and
deployed by the maintainer. The project's engineering discipline is designed
for exactly this workflow — byte-oracle tests pin the wire format, full test
suites gate every image build, and real-client verification signs off
gameplay. Disclosed here for transparency; judge the code on its behavior.

## License

Licensed under the **Apache License, Version 2.0** — see [LICENSE](LICENSE)
and [NOTICE](NOTICE). Note §6: the license grants no rights to the tachyne
name or any trademarks.

## Disclaimer

tachyne is an unofficial, independent project. It is **not** affiliated with,
endorsed, sponsored, or approved by Mojang Studios, Mojang Synergies AB,
Microsoft Corporation, or any of their subsidiaries — the developer and
publisher of Minecraft have no involvement with this project. "Minecraft" is
a trademark of Mojang Synergies AB. This project contains no Minecraft game
code; all game behavior is independently reimplemented, and data tables are
built from openly licensed community datasets (see Credits).
