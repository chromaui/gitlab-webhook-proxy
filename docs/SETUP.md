## Table of Contents

1. [Tools](#tools)
   - [Required](#required)
   - [Recommended](#recommended)
2. [Setup environment variable file](#setup-environment-variable-file)
3. [Creating a test SQS queue](#creating-a-test-sqs-queue)
4. [Starting the GitLab environment](#starting-the-gitlab-environment)
5. [Create project in Chromatic](#create-project-in-chromatic)
6. [Add SQS queue to app in Chromatic](#add-sqs-queue-to-app-in-chromatic)

## Tools

### Required

- [Go](https://golang.org/) - Build and run the application
- A GitLab repository of some sort (this can be a local version or one hosted on gitlab.com)

### Recommended

- [Docker Compose](https://docs.docker.com/compose/) - Spin up a local GitLab environment for testing
- [direnv](https://github.com/direnv/direnv) - Automatically set environment variables based on directories
- [Mockery](https://github.com/vektra/mockery) - Automatically generate mock types for project interfaces

## Setup environment variable file

We're going to be adding environment variables as we go along so copy the `.envrc-example` to `.envrc`.

## Creating a test SQS queue

1. Open the [AWS Management Console](https://us-west-2.console.aws.amazon.com/console/home?region=us-west-2#)
2. Go to the [Simple Queue Service](https://us-west-2.console.aws.amazon.com/sqs/v2/home?region=us-west-2#/queues) page
3. Click `Create queue`
4. Add queue settings
   - Type: `Standard`
   - Name: `gitlabWebhookTest-<your_name>`
   - Configuration:
     - Visibility timeout: `30 Seconds`
     - Delivery delay: `0 Seconds`
     - Receive message wait time: `0 Seconds`
     - Message retention period: `10 Minutes`
     - Maximum message size: `256 KB`
   - Access policy
     - Method: `Basic`
     - Define who can send messages to the queue: `Only the specified AWS accounts, IAM users and roles`
       - Account ARN: `arn:aws:iam::900289697209:user/tunnel` (this is based on the `AWS_ACCESS_KEY_ID` environment value from the `chromatic-index` project in Heroku)
     - Define who can receive messages from the queue: `Only the specified AWS accounts, IAM users and roles`
       - Account ARN: `arn:aws:iam::900289697209:user/gitlab-webhook-proxy-reader` (key and token are in 1Password)
   - Encryption
     - Server-side encryption: `Enabled`
     - Leave the default settings

## Starting the GitLab environment

1. Start up the Docker environment

   ```bash
   # run GitLab in the background (recommended so you don't have to keep your terminal open)
   docker-compose up --detach

   # run GitLab in the foreground
   docker-compose up
   ```

   _Note:_ If you started the environment in detached mode, you can follow the logs with `docker-compose logs --follow` and `ctrl-c` will exit (containers will continue running).

2. Navigate to the GitLab web UI at https://localhost:8929

   _Note:_ This takes a while to load because of how the GitLab container functions. It needs to go through a fairly lengthy setup process but the web UI should load in around 5 minutes or so.

3. Log into the `root` account

   ```
   Username: root
   Password: hichroma
   ```

4. Create test repository

   - Click `New project`
   - Click `Import project`
   - Click `Repo by URL`
   - Add our `simple-app` URL: `https://github.com/chromaui/simple-app.git`
   - Click `Create project`

5. Create access token

   - Click the project dropdown in the top-right corner
   - Click `Edit profile`
   - Click `Access Tokens` in the sidebar
   - Name the token whatever you want
   - Check the `api` scope
   - Click `Create personal access token`
   - Paste the personal access token that appears near the top of the page to your `.envrc` file under `GITLAB_TOKEN`

## Create project in Chromatic

1. Navigate to a Chromatic app (probably dev-chromatic.com or staging-chromatic.com)
2. Click `Add project`
3. Click `Create a project`
4. Project name: `simple-app`
5. Click `Continue`
6. (optional) Run through the setup steps

   - You can do this with the repository we setup in your local GitLab instance

     ```bash
     git clone http://localhost:8929/root/simple-app.git

     # username: root
     # password: hichroma
     ```

   - Then run the setup steps (don't forget to set `CHROMATIC_INDEX_URL` based on the version of Chromatic you're running)

## Add SQS queue to app in Chromatic

Navigate to the `/graphql` endpoint of whatever instance of Chromatic you're using to test and run the following mutation (don't forget to update the `<>` fields):

```
mutation {
  updateApp(id:"<app_id>", input: { sqsQueueUrl:"<your_sqs_queue_url>" }) {
    id
    sqsQueueUrl
  }
}
```
