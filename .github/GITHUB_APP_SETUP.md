# GitHub App Setup for Workflow Authentication

This guide will help you create a GitHub App to generate temporary tokens for the `run-release.yml` workflow.

## Step 1: Create a GitHub App

1. Go to your GitHub repository settings:
   - Navigate to `https://github.com/aditya7k/example-agent/settings/apps`
   - Or: Settings → Developer settings → GitHub Apps → New GitHub App

2. Fill in the required information:
   - **GitHub App name**: `example-agent-workflow` (or any unique name)
   - **Homepage URL**: `https://github.com/aditya7k/example-agent`
   - **Webhook**: Uncheck "Active" (not needed for this use case)

3. Set permissions:
   - Under "Repository permissions":
     - **Contents**: Read and write (if your agent needs to create/modify files)
     - **Issues**: Read and write (if your agent works with issues)
     - **Pull requests**: Read and write (if your agent works with PRs)
     - **Metadata**: Read-only (automatically required)
   - Adjust permissions based on what your agent actually needs

4. Set "Where can this GitHub App be installed?":
   - Select "Only on this account"

5. Click "Create GitHub App"

## Step 2: Generate Private Key

1. After creating the app, scroll down to "Private keys"
2. Click "Generate a private key"
3. A `.pem` file will be downloaded to your computer
4. **Keep this file secure** - it's like a password for your app

## Step 3: Install the GitHub App

1. In your GitHub App settings, click "Install App" in the left sidebar
2. Click "Install" next to your username/organization
3. Select "Only select repositories" and choose `example-agent`
4. Click "Install"

## Step 4: Configure Repository Secrets and Variables

1. Go to your repository settings:
   - Navigate to `https://github.com/aditya7k/example-agent/settings/secrets/actions`

2. Add the App ID as a **variable**:
   - Go to "Variables" tab
   - Click "New repository variable"
   - Name: `APP_ID`
   - Value: Your App ID (found on the GitHub App settings page, e.g., `123456`)

3. Add the Private Key as a **secret**:
   - Go to "Secrets" tab (or go to `https://github.com/aditya7k/example-agent/settings/secrets/actions`)
   - Click "New repository secret"
   - Name: `APP_PRIVATE_KEY`
   - Value: Copy the entire contents of the `.pem` file, including the header and footer lines:
     ```
     -----BEGIN RSA PRIVATE KEY-----
     ... entire key contents ...
     -----END RSA PRIVATE KEY-----
     ```

## Step 5: Test the Workflow

1. Go to Actions tab: `https://github.com/aditya7k/example-agent/actions`
2. Select "Run Release Binary" workflow
3. Click "Run workflow"
4. Check the workflow run logs to verify the token was generated successfully

## Troubleshooting

- **Error: "App ID or Private Key not found"**: Make sure `APP_ID` is set as a variable (not a secret) and `APP_PRIVATE_KEY` is set as a secret
- **Error: "App not installed"**: Make sure you installed the GitHub App on the repository
- **Permission errors**: Review the permissions granted to your GitHub App and add any missing ones

## Security Notes

- The generated token expires after 1 hour by default
- The token only has the permissions you granted to the GitHub App
- Never commit the private key (`.pem` file) to your repository
- The private key should only be stored as a GitHub secret
