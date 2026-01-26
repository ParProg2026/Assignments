<div align="center">
  
# Parallel and Distributed Programming (2026)
## Assignments & reports

![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/ParProg2026/Assignments/.github%2Fworkflows%2FpublishReportsToPages.yml?style=for-the-badge)
[![GitHub Pages](https://img.shields.io/badge/GitHub%20Pages-Deployed-blue?style=for-the-badge)](https://parprog2026.github.io/Assignments)
![LaTeX](https://img.shields.io/badge/LaTeX-Automated-green?style=for-the-badge)

</div>

This repository is the centralized hub for submitting and publishing assignments for the **Parallel and Distributed Programming (2026)** course.

It provides an automated **CI/CD pipeline** that compiles LaTeX reports and publishes the resulting PDFs via **GitHub Pages**, ensuring a consistent, reproducible, and low-friction workflow.

---

## 🚀 Automated Publishing

The repository uses a custom **GitHub Actions workflow** to automatically build and publish reports.

### How it works
- **Trigger**  
  Every push to the `main` branch starts the build process.

- **LaTeX compilation**  
  The workflow recursively searches for files named `report.tex`.

- **Supported directory depth**  
  Reports may be placed up to **two directory levels deep**, for example:
```
Assignments/threeDogs/report.tex
Assignments/threeDogs/report/report.tex
```


- **Deployment**  
Successfully compiled PDFs are deployed to **GitHub Pages** and indexed automatically.

---

## 📂 Directory Structure

To ensure your assignment is compiled correctly, follow this structure exactly:

```
.
├── .github/workflows/ # CI/CD pipeline configuration
├── DogsAndGarden/ # Example assignment directory
│ ├── report.tex # Main LaTeX file (required name)
│ └── tournament.png # Images and assets
└── README.md
```

---
## 🔗 Published Reports

All successfully compiled assignments are available at:

👉 https://parprog2026.github.io/Assignments

From there you can:

* Browse the auto-generated index
* Download individual PDFs
* Verify that your submission compiled correctly
