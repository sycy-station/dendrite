---
title: Contributing
parent: Development
nav_order: 1
permalink: /development/contributing
---

# Contributing to Dendrite

Everyone is welcome to contribute to Dendrite! We aim to make it as easy as
possible to get started.

## Contribution types

We are a small team maintaining a large project. As a result, we cannot merge every feature, even if it
is bug-free and useful, because we then commit to maintaining it indefinitely. We will always accept:
 - bug fixes
 - security fixes (please responsibly disclose via security@matrix.org *before* creating pull requests)

We will accept the following with caveats:
 - documentation fixes, provided they do not add additional instructions which can end up going out-of-date,
   e.g example configs, shell commands.
 - performance fixes, provided they do not add significantly more maintenance burden.
 - additional functionality on existing features, provided the functionality is small and maintainable.
 - additional functionality that, in its absence, would impact the ecosystem e.g spam and abuse mitigations
 - test-only changes, provided they help improve coverage or test tricky code.

The following items are at risk of not being accepted:
 - Configuration or CLI changes, particularly ones which increase the overall configuration surface.

The following items are unlikely to be accepted into a main Dendrite release for now:
 - New MSC implementations.
 - New features which are not in the specification.

## Sign off

We ask that everybody who contributes to this project signs off their contributions, as explained below.

We follow a simple 'inbound=outbound' model for contributions: the act of submitting an 'inbound' contribution means that the contributor agrees to license their contribution under the same terms as the project's overall 'outbound' license - in our case, this is Apache Software License v2 (see [LICENSE](../..//LICENSE)).

In order to have a concrete record that your contribution is intentional and you agree to license it under the same terms as the project's license, we've adopted the same lightweight approach used by the [Linux Kernel](https://www.kernel.org/doc/html/latest/process/submitting-patches.html), [Docker](https://github.com/docker/docker/blob/master/CONTRIBUTING.md), and many other projects: the [Developer Certificate of Origin](https://developercertificate.org/) (DCO). This is a simple declaration that you wrote the contribution or otherwise have the right to contribute it to Matrix:

```
Developer Certificate of Origin
Version 1.1
Copyright (C) 2004, 2006 The Linux Foundation and its contributors.
660 York Street, Suite 102,
San Francisco, CA 94110 USA
Everyone is permitted to copy and distribute verbatim copies of this
license document, but changing it is not allowed.
Developer's Certificate of Origin 1.1
By making a contribution to this project, I certify that:
(a) The contribution was created in whole or in part by me and I
    have the right to submit it under the open source license
    indicated in the file; or
(b) The contribution is based upon previous work that, to the best
    of my knowledge, is covered under an appropriate open source
    license and I have the right under that license to submit that
    work with modifications, whether created in whole or in part
    by me, under the same open source license (unless I am
    permitted to submit under a different license), as indicated
    in the file; or
(c) The contribution was provided directly to me by some other
    person who certified (a), (b) or (c) and I have not modified
    it.
(d) I understand and agree that this project and the contribution
    are public and that a record of the contribution (including all
    personal information I submit with it, including my sign-off) is
    maintained indefinitely and may be redistributed consistent with
    this project or the open source license(s) involved.
```

If you agree to this for your contribution, then all that's needed is to include the line in your commit or pull request comment:

```
Signed-off-by: Your Name <your@email.example.org>
```

Git allows you to add this signoff automatically when using the `-s` flag to `git commit`, which uses the name and email set in your `user.name` and `user.email` git configs.

## Getting up and running

See the [Installation](../installation) section for information on how to build an
instance of Dendrite. You will likely need this in order to test your changes.

## Code style

On the whole, the format as prescribed by `gofmt`, `goimports` etc. is exactly
what we use and expect. Please make sure that you run one of these formatters before
submitting your contribution.

## Comments

Please make sure that the comments adequately explain *why* your code does what it
does. If there are statements that are not obvious, please comment what they do.

We also have some special tags which we use for searchability. These are:

* `// TODO:` for places where a future review, rewrite or refactor is likely required;
* `// FIXME:` for places where we know there is an outstanding bug that needs a fix;
* `// NOTSPEC:` for places where the behaviour specifically does not match what the
  [Matrix Specification](https://spec.matrix.org/) prescribes, along with a description
  of *why* that is the case.

## Linting

We use [golangci-lint](https://github.com/golangci/golangci-lint) to lint Dendrite
which can be executed via:

```bash
golangci-lint run
```

If you are receiving linter warnings that you are certain are spurious and want to
silence them, you can annotate the relevant lines or methods with a `// nolint:`
comment. Please avoid doing this if you can.

## Unit tests

We also have unit tests which we run via:

```bash
DENDRITE_TEST_SKIP_NODB=1 go test --race ./...
```

This only runs SQLite database tests. If you wish to execute Postgres tests as well, you'll either need to
have Postgres installed locally (`createdb` will be used) or have a remote/containerized Postgres instance
available.

To configure the connection to a remote Postgres, you can use the following enviroment variables:

```bash
POSTGRES_USER=postgres
POSTGRES_PASSWORD=yourPostgresPassword
POSTGRES_HOST=localhost
POSTGRES_DB=postgres # the superuser database to use
```

In general, we like submissions that come with tests. Anything that proves that the
code is functioning as intended is great, and to ensure that we will find out quickly
in the future if any regressions happen.

We use the standard [Go testing package](https://gobyexample.com/testing) for this,
alongside some helper functions in our own [`test` package](https://pkg.go.dev/github.com/element-hq/dendrite/test).

## Continuous integration

When a Pull Request is submitted, continuous integration jobs are run automatically
by GitHub actions to ensure that the code builds and works in a number of configurations,
such as different Go versions, using full HTTP APIs and both database engines.
CI will automatically run the unit tests (as above) as well as both of our integration
test suites ([Complement](https://github.com/matrix-org/complement) and
[SyTest](https://github.com/matrix-org/sytest)).

You can see the progress of any CI jobs at the bottom of the Pull Request page, or by
looking at the [Actions](https://github.com/element-hq/dendrite/actions) tab of the Dendrite
repository.

We generally won't accept a submission unless all of the CI jobs are passing. We
do understand though that sometimes the tests get things wrong — if that's the case,
please also raise a pull request to fix the relevant tests!

### Running CI tests locally

To save waiting for CI to finish after every commit, it is ideal to run the
checks locally before pushing, fixing errors first. This also saves other people
time as only so many PRs can be tested at a given time.

To execute what CI tests, first run `./build/scripts/build-test-lint.sh`; this
script will build the code, lint it, and run `go test ./...` with race condition
checking enabled. If something needs to be changed, fix it and then run the
script again until it no longer complains. Be warned that the linting can take a
significant amount of CPU and RAM.

Once the code builds, run [Sytest](https://github.com/matrix-org/sytest)
according to the guide in
[docs/development/sytest.md](https://github.com/element-hq/dendrite/blob/main/docs/development/sytest.md#using-a-sytest-docker-image)
so you can see whether something is being broken and whether there are newly
passing tests.

If these two steps report no problems, the code should be able to pass the CI
tests.

## Picking things to do

If you're new then feel free to pick up an issue labelled [good first
issue](https://github.com/element-hq/dendrite/labels/good%20first%20issue).
These should be well-contained, small pieces of work that can be picked up to
help you get familiar with the code base.

Once you're comfortable with hacking on Dendrite there are issues labelled as
[help wanted](https://github.com/element-hq/dendrite/labels/help-wanted),
these are often slightly larger or more complicated pieces of work but are
hopefully nonetheless fairly well-contained.

We ask people who are familiar with Dendrite to leave the [good first
issue](https://github.com/element-hq/dendrite/labels/good%20first%20issue)
issues so that there is always a way for new people to come and get involved.

## Getting help

For questions related to developing on Dendrite we have a dedicated room on
Matrix [#dendrite-dev:matrix.org](https://matrix.to/#/#dendrite-dev:matrix.org)
where we're happy to help.

For more general questions please use [#dendrite:matrix.org](https://matrix.to/#/#dendrite:matrix.org).
