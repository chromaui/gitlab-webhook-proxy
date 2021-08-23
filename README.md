# gitlab-webhook-proxy

Chromatic has some customers that uses a self-hosted Git host (like GitLab/Bitbucket) and cannot open a public webhook endpoint due to security concerns. In these cases, the Chromatic app can't communicate with their instance to apply build updates for pull requests and various commits.

This project allows for this configuration by pulling messages from a queue and applying those messages to the Git host that's internal to the network in which this container lives.

```
+--------------------------------+
|       Internal Network         |
|                                |
|                                |
|        +------------+          |                   +---------------+
|        |            |          |   Starts build    |               |
|        |  Git Host  +----------+------------------>|   Chromatic   |
|        |            |          |                   |               |
|        +------------+          |                   +-------+-------+
|               ^                |                           |
|               |                |                           |
|               | Posts Updates  |            Posts messages |
|               |                |                           |
|               |                |                           v
|  +------------+------------+   |                   +---------------+
|  |                         |   |  Pulls messages   |               |
|  |  gitlab-webhook-proxy   +---+------------------>|   SQS Queue   |
|  |     (this project)      |   |                   |               |
|  |                         |   |                   +---------------+
|  +-------------------------+   |
+--------------------------------+
```

_diagram created with [ASCIIFlow](https://asciiflow.com/#)_

## Project Structure

```
.
+-- mocks [Interface mocks used for testing]
+-- pkg [Bulk of the logic for interacting with external services]
|   +-- githost [Functions for interacting with a Git host (GitLab/BitBucket)]
|   +-- queue [Functions for interacting with a queue (SQS/PubSub)]
|   +-- types [Global type definitions]
```

## Development

### Setup

If you need to setup your environment for the first time, feel free to follow [the setup doc](docs/SETUP.md).

### Generating mocks

If you add/remove/update any of the interfaces in this project, you'll likely need to update our mocks to keep our tests up-to-date. To do that, simply use `mockery` in the root directory (these should be checked in):

```bash
mockery --all
```
