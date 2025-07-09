#!/bin/bash

# SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

process_json() {
  local json=$1

  echo $json | \
    jq -r '
      ([to_entries[] | select(.value|type=="string") | "\(.key)=\"\(.value)\""] | join(",")) as $labels
      | to_entries[] | select(.value|type=="number")
      | "rsyslog_pstat_\(.key | sub("\\.";"_")){\($labels)} \(.value)"
    ' || { logger -p error -t  process_rsyslog_pstats.sh  "Error processing JSON: $json"; exit 1; }
}

process_line() {
  local line=$1
  local json

  json=$(echo $line | sed -n 's/.*rsyslogd-pstats: //p') || { logger -p error -t  process_rsyslog_pstats.sh  "Error extracting JSON from line: $line"; return 1; }
  process_json "$json"
}

add_comments() {
  local prefix=$1
  local help_and_type=()

  case $prefix in
    *"rsyslog_pstat_submitted")
      help_and_type=('Number of messages submitted' 'counter')
      ;;
    *"processed")
      help_and_type=('Number of messages processed' 'counter')
      ;;
    *"rsyslog_pstat_failed")
      help_and_type=('Number of messages failed' 'counter')
      ;;
    *"rsyslog_pstat_suspended")
      help_and_type=('Number of times suspended' 'counter')
      ;;
    *"rsyslog_pstat_suspended_duration")
      help_and_type=('Time spent suspended' 'counter')
      ;;
    *"rsyslog_pstat_resumed")
      help_and_type=('Number of times resumed' 'counter')
      ;;
    *"rsyslog_pstat_utime")
      help_and_type=('User time used in microseconds' 'counter')
      ;;
    *"rsyslog_pstat_stime")
      help_and_type=('System time used in microsends' 'counter')
      ;;
    *"rsyslog_pstat_maxrss")
      help_and_type=('Maximum resident set size' 'gauge')
      ;;
    *"rsyslog_pstat_minflt")
      help_and_type=('Total minor faults' 'counter')
      ;;
    *"rsyslog_pstat_majflt")
      help_and_type=('Total major faults' 'counter')
      ;;
    *"rsyslog_pstat_inblock")
      help_and_type=('Filesystem input operations' 'counter')
      ;;
    *"rsyslog_pstat_oublock")
      help_and_type=('Filesystem output operations' 'counter')
      ;;
    *"rsyslog_pstat_nvcsw")
      help_and_type=('Voluntary context switches' 'counter')
      ;;
    *"rsyslog_pstat_nivcsw")
      help_and_type=('Involuntary context switches' 'counter')
      ;;
    *"rsyslog_pstat_openfiles")
      help_and_type=('Number of open files' 'counter')
      ;;
    *"rsyslog_pstat_size")
      help_and_type=('Messages currently in queue' 'gauge')
      ;;
    *"rsyslog_pstat_enqueued")
      help_and_type=('Total messages enqueued' 'counter')
      ;;
    *"rsyslog_pstat_full")
      help_and_type=('Times queue was full' 'counter')
      ;;
    *"rsyslog_pstat_discarded_full")
      help_and_type=('Messages discarded due to queue being full' 'counter')
      ;;
    *"rsyslog_pstat_discarded_nf")
      help_and_type=('Messages discarded when queue not full' 'counter')
      ;;
    *"rsyslog_pstat_maxqsize")
      help_and_type=('Maximum size queue has reached' 'gauge')
      ;;
  esac

  if [ ${#help_and_type[@]} -eq 0 ]; then
    return 0
  fi

  comments+="# HELP ${prefix} ${help_and_type[0]}.\n"
  comments+="# TYPE ${prefix} ${help_and_type[1]}"
  echo "$comments"
}

declare -a lines
output_dir="/var/lib/node-exporter/textfile-collector"
output_file="$output_dir/rsyslog_pstats.prom"
output=""
prev_prefix=""

# Create output directory if it does not exist.
mkdir -p "$output_dir"

while IFS= read -r line; do
  if [[ $line == *"rsyslogd-pstats: BEGIN"* ]]; then
    # Start of a new batch, clear the lines array and the output string.
    lines=()
    output=""
    prev_prefix=""
  elif [[ $line == *"rsyslogd-pstats: END"* ]]; then
    # End of a batch, sort the lines array, aggregate the output and write it to a file.
    IFS=$'\n' lines=($(sort <<<"${lines[*]}"))
    for line in "${lines[@]}"; do
      prefix=$(echo "$line" | cut -d'{' -f1) || { logger -p error -t  process_rsyslog_pstats.sh  "Error extracting prefix from line: $line"; exit 1; }
      if [[ "$prefix" != "$prev_prefix" ]]; then
        comment="$(add_comments "$prefix")"
        if [[ -z $comment ]]; then
          # If no help and type comment was added, then the metric is not known. Hence we do not add it
          # to the output.
          continue
        fi
        output+="$comment\n"
        prev_prefix="$prefix"
      fi
      output+="$line\n"
    done
    # Writing to the output prom file has to be done with an atomic operation. This is why we first write to a temporary file
    # and then we move/rename the temporary file to the actual output file.
    echo -e "$output" >> "$output_file.tmp" || { logger -p error -t  process_rsyslog_pstats.sh  "Error writing to temp output file"; exit 1; }
    mv "$output_file.tmp" "$output_file"
  else
    processed_line=$(process_line "$line") || { logger -p error -t  process_rsyslog_pstats.sh  "Error processing line: $line"; exit 1; }
    lines+=("$processed_line")
  fi
done