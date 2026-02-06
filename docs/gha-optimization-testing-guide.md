# GitHub Actions Optimization Testing Guide

**Branch:** `gha-optimizations-registry-cache-test`
**Date:** February 1, 2026
**Purpose:** Test and validate caching optimizations in isolation before production rollout

---

## Overview

This guide explains how to test the three priority optimizations:
1. Registry cache for Docker builds
2. Go module caching for backend tests
3. Docker layer caching for E2E tests

Each optimization has a dedicated test workflow that can be triggered manually to measure performance improvements against baseline.

---

## Test Workflows

### 1. Test: Registry Cache Optimization

**Workflow:** `.github/workflows/test-registry-cache.yml`
**Purpose:** Test Docker build caching using container registry as cache backend
**Baseline:** `components-build-deploy.yml` (current production workflow)

#### How to Run

```bash
# Via GitHub UI:
# 1. Go to Actions tab
# 2. Select "TEST: Registry Cache Optimization"
# 3. Click "Run workflow"
# 4. Configure inputs:
#    - component: Choose which component to test (frontend/backend/operator/claude-runner/state-sync)
#    - test_iteration: Iteration number (1, 2, 3, etc.)
#    - enable_registry_cache: true (test) or false (baseline)

# Via GitHub CLI:
gh workflow run test-registry-cache.yml \
  -f component=frontend \
  -f test_iteration=1 \
  -f enable_registry_cache=true
```

#### Test Procedure

**Iteration 1 (Baseline - GHA cache only):**
```bash
gh workflow run test-registry-cache.yml \
  -f component=frontend \
  -f test_iteration=1 \
  -f enable_registry_cache=false
```
- Expected: ~30-38 min (cold cache)
- This establishes baseline performance

**Iteration 2 (First run with registry cache):**
```bash
gh workflow run test-registry-cache.yml \
  -f component=frontend \
  -f test_iteration=2 \
  -f enable_registry_cache=true
```
- Expected: ~30-38 min (populating registry cache)
- Should be similar to baseline (no cache hit yet)

**Iteration 3 (Second run with registry cache):**
```bash
gh workflow run test-registry-cache.yml \
  -f component=frontend \
  -f test_iteration=3 \
  -f enable_registry_cache=true
```
- Expected: ~5-10 min (cache hit!)
- This should show 75-85% improvement

**Iteration 4 (Verify consistency):**
```bash
gh workflow run test-registry-cache.yml \
  -f component=frontend \
  -f test_iteration=4 \
  -f enable_registry_cache=true
```
- Expected: ~5-10 min (cache hit)
- Confirms cache is working consistently

#### What to Measure

1. **Build Duration:**
   - Check workflow summary for total duration
   - Compare iteration 1 (baseline) vs iteration 3-4 (optimized)

2. **Cache Size:**
   ```bash
   # Check registry cache size in Quay.io
   docker pull quay.io/ambient_code/vteam_frontend:buildcache
   docker images quay.io/ambient_code/vteam_frontend:buildcache

   # Check GHA cache
   gh api repos/{owner}/{repo}/actions/caches --jq '.actions_caches[] | select(.key | contains("frontend"))'
   ```

3. **Artifacts:**
   - Download test results JSON from workflow artifacts
   - Compare `duration_seconds` across iterations

#### Success Criteria

- ✅ First run with registry cache: Similar to baseline (±5%)
- ✅ Second run with registry cache: 75-85% faster than baseline
- ✅ Cache persists across runs (consistent performance)
- ✅ Build output is identical (verify image hash)

---

### 2. Test: Go Module Caching

**Workflow:** `.github/workflows/test-go-module-cache.yml`
**Purpose:** Test Go module and build caching
**Baseline:** `backend-unit-tests.yml` (current production workflow)

#### How to Run

```bash
# Via GitHub UI:
# 1. Go to Actions tab
# 2. Select "TEST: Go Module Caching Optimization"
# 3. Click "Run workflow"
# 4. Configure inputs:
#    - test_iteration: Iteration number
#    - enable_go_cache: true (test) or false (baseline)

# Via GitHub CLI:
gh workflow run test-go-module-cache.yml \
  -f test_iteration=1 \
  -f enable_go_cache=true
```

#### Test Procedure

**Iteration 1 (Baseline - no cache):**
```bash
gh workflow run test-go-module-cache.yml \
  -f test_iteration=1 \
  -f enable_go_cache=false
```
- Expected: ~2.8 min
- Downloads all Go modules

**Iteration 2 (First run with Go cache):**
```bash
gh workflow run test-go-module-cache.yml \
  -f test_iteration=2 \
  -f enable_go_cache=true
```
- Expected: ~2.8 min (populating cache)
- Similar to baseline

**Iteration 3 (Second run with Go cache):**
```bash
gh workflow run test-go-module-cache.yml \
  -f test_iteration=3 \
  -f enable_go_cache=true
```
- Expected: ~2.0 min (cache hit!)
- Should show 30% improvement

**Iteration 4 (Verify consistency):**
```bash
gh workflow run test-go-module-cache.yml \
  -f test_iteration=4 \
  -f enable_go_cache=true
```
- Expected: ~2.0 min (cache hit)
- Confirms cache works consistently

#### What to Measure

1. **Test Duration:**
   - Check workflow summary
   - Compare iteration 1 vs 3-4

2. **Cache Contents:**
   - Check workflow logs for "Cache Status" section
   - Look for cache hit/miss messages

3. **Test Results:**
   - Download HTML test report from artifacts
   - Verify tests pass and coverage is same

#### Success Criteria

- ✅ First run with cache: Similar to baseline (±10%)
- ✅ Second run with cache: 25-35% faster than baseline
- ✅ All tests pass (same results as baseline)
- ✅ Coverage reports are identical

---

### 3. Test: E2E Docker Layer Caching

**Workflow:** `.github/workflows/test-e2e-docker-cache.yml`
**Purpose:** Test Docker layer caching for E2E test image builds
**Baseline:** `e2e.yml` (current production workflow)

#### How to Run

```bash
# Via GitHub UI:
# 1. Go to Actions tab
# 2. Select "TEST: E2E Docker Layer Caching"
# 3. Click "Run workflow"
# 4. Configure inputs:
#    - test_iteration: Iteration number
#    - enable_docker_cache: true (test) or false (baseline)
#    - force_build_all: true (build all) or false (detect changes)

# Via GitHub CLI:
gh workflow run test-e2e-docker-cache.yml \
  -f test_iteration=1 \
  -f enable_docker_cache=true \
  -f force_build_all=true
```

#### Test Procedure

**Iteration 1 (Baseline - no cache, build all):**
```bash
gh workflow run test-e2e-docker-cache.yml \
  -f test_iteration=1 \
  -f enable_docker_cache=false \
  -f force_build_all=true
```
- Expected: ~7-8 min
- Builds all 4 components from scratch

**Iteration 2 (First run with cache, build all):**
```bash
gh workflow run test-e2e-docker-cache.yml \
  -f test_iteration=2 \
  -f enable_docker_cache=true \
  -f force_build_all=true
```
- Expected: ~7-8 min (populating cache)
- Similar to baseline

**Iteration 3 (Second run with cache, build all):**
```bash
gh workflow run test-e2e-docker-cache.yml \
  -f test_iteration=3 \
  -f enable_docker_cache=true \
  -f force_build_all=true
```
- Expected: ~5-6 min (cache hit!)
- Should show 20-30% improvement

**Iteration 4 (Change detection test):**
```bash
# Make a small change to frontend code
echo "// test change" >> components/frontend/src/app/page.tsx
git add components/frontend/src/app/page.tsx
git commit -m "test: trigger frontend change"
git push

gh workflow run test-e2e-docker-cache.yml \
  -f test_iteration=4 \
  -f enable_docker_cache=true \
  -f force_build_all=false
```
- Expected: Only frontend builds, others use cache or pull
- Much faster than full rebuild

#### What to Measure

1. **Workflow Duration:**
   - Total end-to-end time
   - Build phase duration specifically

2. **Component Build Times:**
   - Check logs for individual component build times
   - Compare cache hits vs misses

3. **E2E Test Results:**
   - Verify all Cypress tests pass
   - Check for any deployment issues

#### Success Criteria

- ✅ First run with cache: Similar to baseline (±10%)
- ✅ Second run with cache (force_build_all): 20-30% faster
- ✅ Change detection works: Only changed components rebuild
- ✅ All E2E tests pass (same results as baseline)
- ✅ No deployment regressions

---

## Comparison Methodology

### Step 1: Collect Baseline Data

Run each baseline workflow 3 times to establish performance baseline:

```bash
# Registry cache baseline (use production workflow)
gh workflow run components-build-deploy.yml

# Go cache baseline
gh workflow run backend-unit-tests.yml

# E2E cache baseline
gh workflow run e2e.yml
```

Record the durations from each run.

### Step 2: Run Test Workflows

Execute the test procedure for each optimization (as detailed above).

### Step 3: Download and Analyze Results

```bash
# Download all test result artifacts
gh run download <run-id>

# Or use GitHub UI to download artifacts

# Parse JSON results
jq '.' test-results/*.json

# Calculate improvements
python3 << 'EOF'
import json
import glob

for file in glob.glob('test-results/*.json'):
    with open(file) as f:
        data = json.load(f)
        print(f"{data['test_iteration']}: {data['duration_formatted']} ({data['duration_seconds']}s)")
EOF
```

### Step 4: Create Comparison Report

Create a summary table:

| Test | Baseline | Optimized (1st) | Optimized (2nd) | Improvement |
|------|----------|-----------------|-----------------|-------------|
| Registry Cache (frontend) | 38 min | 37 min | 8 min | 79% |
| Go Module Cache | 2.8 min | 2.7 min | 2.0 min | 29% |
| E2E Docker Cache | 7.2 min | 7.0 min | 5.5 min | 24% |

### Step 5: Validate Output Consistency

**For Registry Cache:**
```bash
# Compare image hashes
docker pull quay.io/ambient_code/vteam_frontend:test-baseline-1
docker pull quay.io/ambient_code/vteam_frontend:test-registry-cache-3

docker inspect quay.io/ambient_code/vteam_frontend:test-baseline-1 --format='{{.Id}}'
docker inspect quay.io/ambient_code/vteam_frontend:test-registry-cache-3 --format='{{.Id}}'

# Hashes won't match due to build timestamps, but layer structure should be identical
docker history quay.io/ambient_code/vteam_frontend:test-baseline-1
docker history quay.io/ambient_code/vteam_frontend:test-registry-cache-3
```

**For Go Cache:**
```bash
# Compare test results
diff baseline-test-report.html optimized-test-report.html

# Check coverage
# Both should have same coverage %
```

**For E2E Cache:**
```bash
# Compare Cypress test results
# All tests should pass in both baseline and optimized runs
```

---

## Troubleshooting

### Registry Cache Issues

**Problem:** Cache not hitting on second run

**Solution:**
1. Check cache was actually created:
   ```bash
   docker pull quay.io/ambient_code/vteam_frontend:buildcache
   ```
2. Verify Quay.io credentials are valid
3. Check Docker buildx is using correct cache scope

**Problem:** Build is slower with cache

**Solution:**
1. First run will always be slower (cache population)
2. Check network speed - registry cache requires good bandwidth
3. Verify cache mode is `mode=max` (exports all layers)

### Go Cache Issues

**Problem:** Cache not improving performance

**Solution:**
1. Check `go.sum` hasn't changed
2. Verify cache key in logs
3. Check Actions cache size limit (10GB repo limit)

**Problem:** Tests fail with cache enabled

**Solution:**
1. Clear cache and retry
2. Check for test flakiness unrelated to caching
3. Verify Go version matches

### E2E Cache Issues

**Problem:** Images not building with cache

**Solution:**
1. Check buildx is installed correctly
2. Verify cache scopes are unique per component
3. Check disk space (buildx cache can be large)

**Problem:** E2E tests fail after caching

**Solution:**
1. This suggests a real issue, not cache-related
2. Check if images are correct: `docker images | grep e2e-test`
3. Verify image tags match what Kind expects
4. Check deployment logs

---

## Monitoring Cache Performance

### GitHub Actions Cache Usage

```bash
# List all caches
gh api repos/{owner}/{repo}/actions/caches

# Filter by scope
gh api repos/{owner}/{repo}/actions/caches --jq '.actions_caches[] | select(.key | contains("frontend"))'

# Check total cache size
gh api repos/{owner}/{repo}/actions/caches --jq '[.actions_caches[].size_in_bytes] | add'
```

### Registry Cache Usage

```bash
# Check Quay.io for buildcache tags
curl -X GET https://quay.io/api/v1/repository/ambient_code/vteam_frontend/tag/ \
  -H "Authorization: Bearer $QUAY_TOKEN" | jq '.tags[] | select(.name | contains("buildcache"))'

# Check image size
docker pull quay.io/ambient_code/vteam_frontend:buildcache
docker images quay.io/ambient_code/vteam_frontend:buildcache
```

---

## Next Steps After Testing

### If Tests Pass

1. **Merge to main:**
   ```bash
   git push origin gha-optimizations-registry-cache-test
   # Create PR
   gh pr create --title "Add caching optimizations to GHA workflows" \
     --body "See docs/gha-optimization-testing-guide.md for test results"
   ```

2. **Update production workflows:**
   - Apply registry cache config to `components-build-deploy.yml`
   - Apply Go cache to `backend-unit-tests.yml`
   - Apply Docker cache to `e2e.yml`

3. **Monitor in production:**
   - Watch first few runs for issues
   - Track cache hit rates
   - Measure actual savings

### If Tests Fail

1. **Document failure:**
   - What failed?
   - Error messages?
   - Reproducible?

2. **Adjust configuration:**
   - Try different cache scopes
   - Adjust cache modes
   - Test with smaller components first

3. **Re-test:**
   - Iterate on test workflows
   - Document changes
   - Repeat testing procedure

---

## Cost Analysis

### Registry Cache Storage Cost

- **Cache size per component:** ~500MB-1GB
- **Total for 5 components:** ~2.5-5GB
- **Quay.io storage cost:** ~$0.10/GB/month
- **Monthly cost:** ~$0.25-$0.50

### GHA Cache Usage

- **Current usage:** Check with `gh api repos/{owner}/{repo}/actions/caches`
- **Limit per repo:** 10GB
- **Cost:** Free (included in GitHub plan)

### Compute Time Savings

- **Monthly baseline:** 17,884 min ($143)
- **Projected savings:** 38% reduction
- **Monthly optimized:** ~11,000 min ($88)
- **Net savings:** $55/month after storage costs

**ROI:** ~132x (savings vs storage cost)

---

## Appendix: Quick Reference Commands

### Trigger All Tests (Full Suite)

```bash
# Registry cache test - all components
for component in frontend backend operator claude-runner state-sync; do
  gh workflow run test-registry-cache.yml \
    -f component=$component \
    -f test_iteration=1 \
    -f enable_registry_cache=true
done

# Go cache test
gh workflow run test-go-module-cache.yml \
  -f test_iteration=1 \
  -f enable_go_cache=true

# E2E cache test
gh workflow run test-e2e-docker-cache.yml \
  -f test_iteration=1 \
  -f enable_docker_cache=true \
  -f force_build_all=true
```

### Download All Test Results

```bash
# List recent workflow runs
gh run list --workflow=test-registry-cache.yml --limit 10

# Download artifacts from specific run
gh run download <run-id>

# Or download all from latest run
gh run download $(gh run list --workflow=test-registry-cache.yml --limit 1 --json databaseId --jq '.[0].databaseId')
```

### Compare Performance

```bash
# Create comparison script
cat > analyze-results.sh << 'EOF'
#!/bin/bash
echo "Test Results Comparison"
echo "======================="
for file in test-results/*.json; do
  iter=$(jq -r '.test_iteration' "$file")
  dur=$(jq -r '.duration_formatted' "$file")
  enabled=$(jq -r '.registry_cache_enabled // .go_cache_enabled // .docker_cache_enabled' "$file")
  echo "Iteration $iter (cache=$enabled): $dur"
done | sort -V
EOF

chmod +x analyze-results.sh
./analyze-results.sh
```

---

**Document Version:** 1.0
**Last Updated:** February 1, 2026
**Maintainer:** Platform DevOps Team
