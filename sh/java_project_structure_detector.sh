#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Output directory and file
OUTPUT_DIR=".asap"
OUTPUT_FILE="$OUTPUT_DIR/project.toml"

print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Detect project type
detect_project_type() {
    if [[ -f "pom.xml" ]]; then
        echo "maven"
    elif [[ -f "build.gradle" || -f "build.gradle.kts" || -f "settings.gradle" || -f "settings.gradle.kts" ]]; then
        echo "gradle"
    else
        print_error "No Maven or Gradle project detected"
        exit 1
    fi
}

# Get Gradle command
get_gradle_cmd() {
    if [[ -f "./gradlew" ]]; then
        echo "./gradlew"
    elif command -v gradle &> /dev/null; then
        echo "gradle"
    else
        print_error "Neither ./gradlew nor gradle command found"
        exit 1
    fi
}

# Get Maven command
get_maven_cmd() {
    if [[ -f "./mvnw" ]]; then
        echo "./mvnw"
    elif command -v mvn &> /dev/null; then
        echo "mvn"
    else
        print_error "Neither ./mvnw nor mvn command found"
        exit 1
    fi
}

# Get build tool version
get_build_tool_version() {
    local tool_type="$1"
    local tool_cmd="$2"

    if [[ "$tool_type" == "gradle" ]]; then
        $tool_cmd --version --console=plain -q 2>/dev/null | grep "^Gradle" | sed 's/Gradle \([0-9.]*\).*/\1/' || echo "unknown"
    elif [[ "$tool_type" == "maven" ]]; then
        $tool_cmd --version 2>/dev/null | grep "^Apache Maven" | sed 's/.* \([0-9.]*\).*/\1/' || echo "unknown"
    else
        echo "unknown"
    fi
}

# Get list of all projects
get_projects_list() {
    local gradle_cmd="$1"
    $gradle_cmd projects --console=plain -q 2>/dev/null | grep "^Project" | sed "s/Project '\(.*\)'/\1/" || {
        print_warning "Could not get projects list"
        echo ":"
    }
}

# Create analysis script for a specific project
create_analysis_script() {
    local project="$1"
    local temp_script=$(mktemp)

    cat > "$temp_script" << EOF
if (project.path == '$project') {
    def info = [:]
    info.name = project.name
    info.path = project.path
    info.projectDir = project.projectDir.absolutePath
    info.buildDir = project.buildDir.absolutePath
    info.buildFile = project.buildFile.absolutePath

    def sourceDirs = []
    def resourceDirs = []
    def testSourceDirs = []
    def testResourceDirs = []

    // Check for Java/Kotlin plugins
    def hasJavaPlugin = project.plugins.hasPlugin('java') ||
                       project.plugins.hasPlugin('java-library') ||
                       project.plugins.hasPlugin('application') ||
                       project.plugins.hasPlugin('org.springframework.boot')

    def hasKotlinPlugin = project.plugins.hasPlugin('org.jetbrains.kotlin.jvm') ||
                         project.plugins.hasPlugin('kotlin')

    if (hasJavaPlugin || hasKotlinPlugin) {
        if (project.hasProperty('sourceSets')) {
            // Main source sets
            if (hasJavaPlugin && project.sourceSets.main.hasProperty('java')) {
                project.sourceSets.main.java.srcDirs.each {
                    if (it.exists()) sourceDirs << it.absolutePath
                }
            }
            if (hasKotlinPlugin && project.sourceSets.main.hasProperty('kotlin')) {
                project.sourceSets.main.kotlin.srcDirs.each {
                    if (it.exists()) sourceDirs << it.absolutePath
                }
            }
            project.sourceSets.main.resources.srcDirs.each {
                if (it.exists()) resourceDirs << it.absolutePath
            }

            // Test source sets
            if (hasJavaPlugin && project.sourceSets.test.hasProperty('java')) {
                project.sourceSets.test.java.srcDirs.each {
                    if (it.exists()) testSourceDirs << it.absolutePath
                }
            }
            if (hasKotlinPlugin && project.sourceSets.test.hasProperty('kotlin')) {
                project.sourceSets.test.kotlin.srcDirs.each {
                    if (it.exists()) testSourceDirs << it.absolutePath
                }
            }
            project.sourceSets.test.resources.srcDirs.each {
                if (it.exists()) testResourceDirs << it.absolutePath
            }
        }
    }

    info.sourceDirs = sourceDirs.unique()
    info.resourceDirs = resourceDirs.unique()
    info.testSourceDirs = testSourceDirs.unique()
    info.testResourceDirs = testResourceDirs.unique()

    println "###PROJECT_INFO_START###"
    println groovy.json.JsonOutput.toJson(info)
    println "###PROJECT_INFO_END###"
}
EOF

    echo "$temp_script"
}

# Run analysis for a specific project
run_project_analysis() {
    local gradle_cmd="$1"
    local project="$2"
    local init_script="$3"

    if [[ "$project" == ":" ]]; then
        $gradle_cmd help --init-script "$init_script" --console=plain -q 2>/dev/null
    else
        $gradle_cmd ":$project:help" --init-script "$init_script" --console=plain -q 2>/dev/null
    fi
}

# Analyze Gradle project
analyze_gradle_project() {
    local gradle_cmd="$1"
    local temp_file=$(mktemp)

    print_info "Analyzing Gradle project..."

    # Get all projects
    local projects=$(get_projects_list "$gradle_cmd")

    if [[ -z "$projects" ]]; then
        print_warning "No projects found, using fallback method"
        rm -f "$temp_file"
        return 1
    fi

    print_info "Found projects: $projects"

    # Analyze each project individually
    for project in $projects; do
        print_info "Analyzing project: $project"
        local analysis_script=$(create_analysis_script "$project")
        local output=$(run_project_analysis "$gradle_cmd" "$project" "$analysis_script")

        # Extract JSON from output
        echo "$output" | sed -n '/###PROJECT_INFO_START###/,/###PROJECT_INFO_END###/p' | grep -v "###PROJECT_INFO" >> "$temp_file"

        rm -f "$analysis_script"
    done

    echo "$temp_file"
}

# Analyze Maven project
analyze_maven_project() {
    local maven_cmd="$1"
    local temp_file=$(mktemp)

    print_info "Analyzing Maven project..."

    # Get project list with exec:exec
    $maven_cmd help:evaluate -Dexpression=project.modules -q -DforceStdout 2>/dev/null > /dev/null || true

    # Extract project information using Maven help plugin
    cat > "$temp_file" << 'EOF'
###PROJECT_INFO_START###
EOF

    # Get root project info
    local root_name=$($maven_cmd help:evaluate -Dexpression=project.artifactId -q -DforceStdout 2>/dev/null || echo "root")
    local root_dir=$(pwd)
    local root_build_dir=$($maven_cmd help:evaluate -Dexpression=project.build.directory -q -DforceStdout 2>/dev/null || echo "$root_dir/target")

    # Write root project JSON
    cat >> "$temp_file" << ROOTEOF
{"name":"$root_name","path":":","projectDir":"$root_dir","buildDir":"$root_build_dir","sourceDirs":[],"resourceDirs":[],"testSourceDirs":[],"testResourceDirs":[]}
ROOTEOF

    # Find all pom.xml files for modules
    find . -name "pom.xml" -not -path "*/target/*" | while read -r pom; do
        local module_dir=$(dirname "$pom")
        if [[ "$module_dir" != "." ]]; then
            pushd "$module_dir" > /dev/null 2>&1
            local module_name=$($maven_cmd help:evaluate -Dexpression=project.artifactId -q -DforceStdout 2>/dev/null || basename "$module_dir")
            local module_build_dir=$($maven_cmd help:evaluate -Dexpression=project.build.directory -q -DforceStdout 2>/dev/null || echo "$module_dir/target")
            popd > /dev/null 2>&1

            echo "{\"name\":\"$module_name\",\"path\":\":$module_name\",\"projectDir\":\"$(cd "$module_dir" && pwd)\",\"buildDir\":\"$module_build_dir\",\"sourceDirs\":[],\"resourceDirs\":[],\"testSourceDirs\":[],\"testResourceDirs\":[]}" >> "$temp_file"
        fi
    done

    cat >> "$temp_file" << 'EOF'
###PROJECT_INFO_END###
EOF

    echo "$temp_file"
}

# Parse project info and write TOML
write_toml_file() {
    local temp_file="$1"
    local project_type="$2"
    local build_tool_version="$3"

    print_info "Writing project information to $OUTPUT_FILE..."

    # Create output directory
    mkdir -p "$OUTPUT_DIR"

    # Start TOML file
    cat > "$OUTPUT_FILE" << EOF
# Project Structure Configuration
# Generated: $(date '+%Y-%m-%d %H:%M:%S')
# Project Type: $project_type

[project]
type = "$project_type"
root = "$(pwd)"
build_tool_version = "$build_tool_version"

EOF

    # Extract and parse JSON data
    local in_section=false
    while IFS= read -r line; do
        if [[ "$line" == "###PROJECT_INFO_START###" ]]; then
            in_section=true
            continue
        elif [[ "$line" == "###PROJECT_INFO_END###" ]]; then
            in_section=false
            continue
        fi

        if [[ "$in_section" == true ]] && [[ -n "$line" ]]; then
            # Parse JSON using Python if available, otherwise use basic parsing
            if command -v python3 &> /dev/null; then
                python3 << PYEOF
import json
import sys
import os

try:
    data = json.loads('''$line''')

    # Write module section
    print("[[modules]]")
    print(f"name = \"{data['name']}\"")
    print(f"path = \"{data['path']}\"")
    print(f"project_dir = \"{data['projectDir']}\"")
    print(f"build_dir = \"{data['buildDir']}\"")
    print(f"build_file = \"{data['buildFile']}\"")

    # Only include existing directories
    source_dirs = [d for d in data.get('sourceDirs', []) if os.path.exists(d)]
    resource_dirs = [d for d in data.get('resourceDirs', []) if os.path.exists(d)]
    test_source_dirs = [d for d in data.get('testSourceDirs', []) if os.path.exists(d)]
    test_resource_dirs = [d for d in data.get('testResourceDirs', []) if os.path.exists(d)]

    print(f"source_dirs = {json.dumps(source_dirs)}")
    print(f"resource_dirs = {json.dumps(resource_dirs)}")
    print(f"test_source_dirs = {json.dumps(test_source_dirs)}")
    print(f"test_resource_dirs = {json.dumps(test_resource_dirs)}")
    print()

except Exception as e:
    print(f"# Error parsing: {e}", file=sys.stderr)
PYEOF
            else
                # Basic fallback without JSON parsing
                echo "# Module detected (install python3 for detailed parsing)"
                echo
            fi >> "$OUTPUT_FILE"
        fi
    done < "$temp_file"

    rm -f "$temp_file"
}

# Manual fallback discovery
manual_discovery() {
    local project_type="$1"
    local build_tool_version="$2"

    print_warning "Using manual discovery fallback..."

    mkdir -p "$OUTPUT_DIR"

    cat > "$OUTPUT_FILE" << EOF
# Project Structure Configuration (Manual Discovery)
# Generated: $(date '+%Y-%m-%d %H:%M:%S')
# Project Type: $project_type

[project]
type = "$project_type"
root = "$(pwd)"
build_tool_version = "$build_tool_version"

EOF

    # Find all source directories
    local module_count=0
    find . -type d -name "src" -not -path "*/target/*" -not -path "*/build/*" -not -path "*/.git/*" | sort | while read -r src_dir; do
        local module_dir=$(dirname "$src_dir")
        local module_name=$(basename "$module_dir")

        if [[ "$module_dir" == "." ]]; then
            module_name="root"
            module_dir=$(pwd)
        else
            module_dir=$(cd "$module_dir" && pwd)
        fi

        echo "[[modules]]" >> "$OUTPUT_FILE"
        echo "name = \"$module_name\"" >> "$OUTPUT_FILE"
        echo "project_dir = \"$module_dir\"" >> "$OUTPUT_FILE"

        # Detect build file
        if [[ -f "$module_dir/build.gradle" ]] || [[ -f "$module_dir/build.gradle.kts" ]]; then
            echo "build_file = \"$module_dir/build.gradle\"" >> "$OUTPUT_FILE"
        elif [[ -f "$module_dir/pom.xml" ]]; then
            echo "build_file = \"$module_dir/pom.xml\"" >> "$OUTPUT_FILE"
        fi

        # Find source directories (only existing ones)
        local main_java="$module_dir/src/main/java"
        local main_kotlin="$module_dir/src/main/kotlin"
        local main_resources="$module_dir/src/main/resources"
        local test_java="$module_dir/src/test/java"
        local test_kotlin="$module_dir/src/test/kotlin"
        local test_resources="$module_dir/src/test/resources"

        echo -n "source_dirs = [" >> "$OUTPUT_FILE"
        local first=true
        for dir in "$main_java" "$main_kotlin"; do
            if [[ -d "$dir" ]]; then
                [[ "$first" == false ]] && echo -n ", " >> "$OUTPUT_FILE"
                echo -n "\"$dir\"" >> "$OUTPUT_FILE"
                first=false
            fi
        done
        echo "]" >> "$OUTPUT_FILE"

        [[ -d "$main_resources" ]] && echo "resource_dirs = [\"$main_resources\"]" >> "$OUTPUT_FILE" || echo "resource_dirs = []" >> "$OUTPUT_FILE"

        echo -n "test_source_dirs = [" >> "$OUTPUT_FILE"
        first=true
        for dir in "$test_java" "$test_kotlin"; do
            if [[ -d "$dir" ]]; then
                [[ "$first" == false ]] && echo -n ", " >> "$OUTPUT_FILE"
                echo -n "\"$dir\"" >> "$OUTPUT_FILE"
                first=false
            fi
        done
        echo "]" >> "$OUTPUT_FILE"

        [[ -d "$test_resources" ]] && echo "test_resource_dirs = [\"$test_resources\"]" >> "$OUTPUT_FILE" || echo "test_resource_dirs = []" >> "$OUTPUT_FILE"

        # Detect build directory
        if [[ -d "$module_dir/target" ]]; then
            echo "build_dir = \"$module_dir/target\"" >> "$OUTPUT_FILE"
        elif [[ -d "$module_dir/build" ]]; then
            echo "build_dir = \"$module_dir/build\"" >> "$OUTPUT_FILE"
        fi

        echo >> "$OUTPUT_FILE"
    done
}

# Main execution
main() {
    print_info "Project Structure Analyzer"
    print_info "=========================="

    local project_type=$(detect_project_type)
    print_success "Detected $project_type project"

    local build_tool_version="unknown"
    local temp_file=""

    if [[ "$project_type" == "gradle" ]]; then
        local gradle_cmd=$(get_gradle_cmd)
        print_success "Using Gradle: $gradle_cmd"
        build_tool_version=$(get_build_tool_version "gradle" "$gradle_cmd")
        temp_file=$(analyze_gradle_project "$gradle_cmd") || {
            manual_discovery "$project_type" "$build_tool_version"
            print_success "Project information saved to $OUTPUT_FILE"
            return 0
        }
    elif [[ "$project_type" == "maven" ]]; then
        local maven_cmd=$(get_maven_cmd)
        print_success "Using Maven: $maven_cmd"
        build_tool_version=$(get_build_tool_version "maven" "$maven_cmd")
        temp_file=$(analyze_maven_project "$maven_cmd") || {
            manual_discovery "$project_type" "$build_tool_version"
            print_success "Project information saved to $OUTPUT_FILE"
            return 0
        }
    fi

    write_toml_file "$temp_file" "$project_type" "$build_tool_version"

    print_success "Project information saved to $OUTPUT_FILE"
    print_info "Summary:"
    grep -c "^\[\[modules\]\]" "$OUTPUT_FILE" | xargs -I {} echo "  - {} module(s) detected"
}

# Show help
show_help() {
    cat << EOF
Project Structure Analyzer

Usage: $0 [OPTIONS]

Options:
  -h, --help    Show this help message

Description:
  Analyzes Gradle or Maven projects and generates a project.toml file
  in the ./asap/ directory containing:

  - Project type (Gradle/Maven)
  - Module/subproject names
  - Source directories (main and test)
  - Resource directories (main and test)
  - Build directories

  The script automatically detects project type and uses the appropriate
  build tool (./gradlew, gradle, ./mvnw, or mvn).

Output:
   .asap/project.toml - TOML configuration file with project structure

Requirements:
  - Run in the root directory of a Gradle or Maven project
  - Python 3 (optional, for better JSON parsing)
EOF
}

# Parse arguments
case "${1:-}" in
    -h|--help)
        show_help
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac