# Nit - AI generate code reviews

[![Go Reference](https://pkg.go.dev/badge/github.com/PuerkitoBio/goquery.svg)](https://pkg.go.dev/github.com/evanmcneely/nit)

Nit is a web server that responds to Github webhook events. It will generate and post code reviews to your pull requests and respond to comments in the thread. This is meant to be a tool to help developers improve the quality of their pull request before requesting review from another dev. Common nit-picky feedback is the goal, but it can suggest improvements, alternative solutions and larger refactors. The feedback can be noisy (hoping to cut down on that) but some good outcomes have emerged from this additional layer of oversight. This tool is in use today at [Leadpages](https://www.leadpages.com).

## How to Use

You will need to deploy this project to a server - which ever way you like to do that - you do you. The entry point is `cmd/server/main.go`.

Add a webhook to the repository you would like generated reviews.

1. Visit Settings > Webhooks > Add webhook
2. You will need to authenticate with Github in order to perform this action
3. Configure the webhook
   - Payload URL = `<yourhostname>/webhooks/github`
   - Content type = `application/json`
   - Secret = (optional/recommended) generate a webhook secret and write it down
   - Which events would you like to trigger this webhook? = select individual events - `Pull requests` and `Pull request review comments`
4. Click "Add webhook" when ready

Create a Github fine-grained access token for your account. I recommend creating the token at the Organization level if possible. Review will be posted on behalf of the account that the token belongs too.

1. In your accounts personal settings > Developer settings > Personal access tokens > Fine-grained access tokens > Generate new token
2. Generate a token with access to the repository you created the webhook in. The specific permissions you need are:
   - Read and write access to pull requests
   - Read access to code and metadata
3. Click "Generate token" when ready

Customize your server environment variables. You can edit the `config.yaml` file directly or set the environment variables yourself - which ever way you like to do that - you do you.

To create environment variables, the config setting ...

```yaml
app:
  webhookSecret: 1234
```

...can be overridden by setting an environment variable with the name `NIT_APP_WEBHOOKSECRET`.

### Configuration

The behaviour of the app can be configured from the `config.yaml` file or with these environment variables.

`NIT_CONFIG_OPTIN`

- Whether or not pull request reviews need to be opted into.
- If `true`, the string `ai-review:please` must be in the pull request description for a review to be generated. If `false`, the string `ai-review:ignore` will cause the pull request to be ignored. The default is `true`.

`NIT_CONFIG_NAME`

- The name of the account the fine-grained access token belongs too. If I created the token, the name would be `evanmcneely`.
- The default is `nit`.

## Development

### Add a new service provider

Adding new AI service providers is as simple implementing the `AIProvider` interface. Every implementation would ideally handle a `completionRequest` to use a "Good" or "Cheap" model. Good being whatever the best model in the line up is in terms of "reasoning" ability, and Cheap being the cost effective one. If the response format is "JSON", the `completionResponse` must be valid JSON or nothing will work.
