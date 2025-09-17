#!/bin/bash

# Setup script for pushing srake to GitHub (nishad/srake)

echo "🍶🧬 Setting up srake for GitHub..."

# 1. Initialize git if needed (already done based on git status output)
echo "📝 Creating initial commit..."
git add .
git commit -F INITIAL_COMMIT_MESSAGE.md

# 2. Add GitHub remote
echo "🔗 Adding GitHub remote..."
git remote add origin https://github.com/nishad/srake.git

# 3. Set main branch
echo "🌿 Setting main branch..."
git branch -M main

# 4. Create and push tag
echo "🏷️  Creating alpha tag..."
git tag -a v0.0.1-alpha -m "Initial alpha release"

# 5. Push to GitHub
echo "🚀 Pushing to GitHub..."
git push -u origin main
git push origin v0.0.1-alpha

echo ""
echo "✅ Done! Your repository is now live at:"
echo "   https://github.com/nishad/srake"
echo ""
echo "📊 Next steps:"
echo "   1. Go to https://github.com/nishad/srake/settings"
echo "   2. Add description: 'High-performance streaming processor for NCBI SRA metadata 🍶🧬'"
echo "   3. Add topics: bioinformatics, sra, ncbi, genomics, golang, sqlite"
echo "   4. Set website (optional): Your lab/personal site"
echo "   5. Enable Issues and Discussions"
echo ""
echo "🏷️  To create a GitHub release:"
echo "   1. Go to https://github.com/nishad/srake/releases/new"
echo "   2. Choose tag: v0.0.1-alpha"
echo "   3. Title: 'v0.0.1-alpha - Initial Alpha Release'"
echo "   4. Check 'Set as pre-release'"
echo "   5. Publish!"

# Clean up
rm INITIAL_COMMIT_MESSAGE.md
rm SETUP_GITHUB.sh