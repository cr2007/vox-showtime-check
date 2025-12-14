# Contributing Guide

Thank you for your interest in contributing! This guide will help you get started and contribute effectively.

## Contribution Flow

1. **Create an Issue**
	 - Open a [new issue](https://github.com/cr2007/vox-showtime-check/issues/new) describing your bug or feature idea.

2. **Fork & Clone the Repository**
	 - [Fork](https://github.com/cr2007/vox-showtime-check/fork) the repo
	 - Clone your forked repository:
		 ```sh
		 git clone https://github.com/<your-username>/vox-showtime-check.git
		 cd vox-showtime-check
		 ```

3. **Set Up Your Development Environment**
	 - Ensure you have [Go 1.18+](https://go.dev/dl/) installed.
	 - Install dependencies:
		 ```sh
		 go mod tidy
		 ```
	 - Copy or create a `.env` file with the required variables:
		 ```env
		 SHOWTIMES_URL=https://uae.voxcinemas.com/movies/<movie>
		 NTFY_TOPIC=my-ntfy-topic
		 ```
	 - Run the project locally to verify setup:
		 ```sh
		 go run main.go
		 ```

4. **Create a Branch & Make Changes**
	 - Create a new branch for your work:
		 ```sh
		 git checkout -b <issue-number>-patch
		 ```
	 - Make your changes.
	 - Test your changes by running the project.

5. **Commit & Push**
	 - Use [Conventional Commits](https://www.conventionalcommits.org) for commit messages (e.g., `feat: add new notification type`).
	 - Push your branch to your fork:
		 ```sh
		 git push origin <branch-name>
		 ```

6. **Open a Pull Request**
	 - Open a [Pull Request (PR)](https://github.com/cr2007/vox-showtime-check/compare) to the `main` branch.
	 - Link the related issue in your PR description and provide a clear summary of your changes.

---

Thank you for helping improve this project! If you have any questions, open an issue or join the discussion.
