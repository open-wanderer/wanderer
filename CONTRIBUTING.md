# Contributing

Thank you for your interest in contributing to `wanderer`. We appreciate all contributions, whether they are bug fixes, documentation updates, translations, or new features.

## Before you start

Before working on a significant change, please:

- Open an issue,
- start a discussion
- or contact us via our [Discord server](https://discord.gg/USSEBY98CP) on the `#dev` channel.

This helps us avoid duplicate work and ensures that your changes align with the project's direction.

For small bug or typo fixes, documentation updates, translations, or clearly isolated improvements, opening a pull request directly is usually fine.

Please follow our [local development guide](https://wanderer.to/develop/local-development/).

## Pull Request Target Branch

Please open pull requests only against the `main` branch.

Pull requests targeting release branches, development branches, or unrelated branches may be closed without review.

## Keep pull requests atomic

Please keep pull requests as small and focused as possible.

A good pull request should address one specific topic, such as:

- One bug fix
- One isolated feature
- One documentation improvement
- One dependency update
- One refactoring

Please avoid combining unrelated changes in the same pull request. For example, do not combine a bug fix with formatting changes, dependency updates, or other refactorings.

Smaller pull requests are easier to review, test, and merge.

## Describe the change clearly

Every pull request should include a clear description of the change.

- What was changed
- Why the change was needed
- How the change was tested
- Any known limitations or side effects.

For UI changes, please include screenshots when helpful.

## Bug fixes and reproduction steps

When fixing a bug, please describe how it can be reproduced from an end-user perspective.

Useful reproduction steps explain the actions a user takes in the application and the problem they observe.

Avoid relying solely on artificial or highly technical steps, such as direct API calls to internal endpoints, unless the issue is specifically related to the API or cannot be reasonably reproduced through the user interface.

## AI-assisted contributions

Using AI tools for assistance is allowed. While AI tools can be helpful, contributors are expected to treat AI-assisted changes like any other code they submit. They should understand, carefully review, and test the changes before opening a pull request.

If you used AI to implement a feature, improve existing functionality, or fix an issue, your pull request should clearly explain the problem or improvement, the intended user-facing behavior, and how the change was tested.

To keep review work manageable, we may close pull requests that appear to be mainly AI-generated or submitted in large numbers without clear evidence that the contributor has reviewed, tested, and understood the changes.

## Testing

Please test your changes before opening a pull request.

If automated tests exist for the affected area, please run them. If no automated tests exist, describe the manual testing you performed.

A useful test description can include the following:

- Operating system/browser/environment
- Relevant configuration
- Exact steps tested
- Expected and actual results

## Breaking changes and migrations

If your PR introduces breaking changes or requires migration, clearly state this in the pull request description.

## Security Issues

Please do not report security vulnerabilities through public issues or pull requests.

If you believe you have found one, please contact the maintainers privately first. You can reach us via our [Discord server](https://discord.gg/USSEBY98CP).

## Reviews

Maintainers may request changes, additional tests, a smaller scope, or a different implementation approach.

Please keep discussions constructive and focused on the code and the user-facing behavior.

Thank you for helping improve the project.