# gitlab-webhook-proxy

This repo serves as an example webhook service to processing [webhook messages](https://www.chromatic.com/docs/integrations#custom-webhooks) from Chromatic.

## Adding a webhook integration to Chromatic

[This doc](https://www.chromatic.com/docs/integrations#custom-webhooks) touches on how to add custom webhook integration to your project. Since Chromatic may not have have access to the GitLab project ID, it's best to include that in your webhook URL so you can use it here.

Example:

```
https://webhook.chromatic.com/webhook?repoId=123
```

## Environment variables

| Variable | Description                                    |
| -------- | ---------------------------------------------- |
| REST_API | Your GitLab API path                           |
| TOKEN    | Your GitLab API token                          |
| PORT     | The desired port this proxy service should use |
