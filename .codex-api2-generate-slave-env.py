#!/usr/bin/env python3
"""Create the api2 runtime environment from a master container env stream."""

import os
import re
import sys
import tempfile


def replace_authority(value: str, target: str, variable: str) -> str:
    match = re.match(r"^([A-Za-z][A-Za-z0-9+.-]*://[^/?#]*@)[^/?#]+(.*)$", value)
    if match is None:
        raise ValueError(f"{variable} has an unsupported connection URL")
    return f"{match.group(1)}{target}{match.group(2)}"


def main() -> None:
    if len(sys.argv) != 2:
        raise SystemExit("usage: generate-slave-env.py DESTINATION")

    master_env: dict[str, str] = {}
    for raw_line in sys.stdin:
        line = raw_line.rstrip("\n")
        if "=" not in line:
            continue
        key, value = line.split("=", 1)
        master_env[key] = value

    for key in ("SQL_DSN", "REDIS_CONN_STRING", "SESSION_SECRET", "CRYPTO_SECRET"):
        if not master_env.get(key):
            raise SystemExit(f"master runtime is missing required {key}")

    sql_dsn = replace_authority(
        master_env["SQL_DSN"], "host.docker.internal:15432", "SQL_DSN"
    )
    redis_connection = replace_authority(
        master_env["REDIS_CONN_STRING"], "host.docker.internal:16379", "REDIS_CONN_STRING"
    )

    inherited_keys = (
        "SESSION_SECRET",
        "CRYPTO_SECRET",
        "TZ",
        "ERROR_LOG_ENABLED",
        "BATCH_UPDATE_ENABLED",
        "ENABLE_METRICS",
        "MONITORING_ENABLED",
        "OBSERVABILITY_EVENT_LOG_ENABLED",
        "METRICS_PORT",
        "METRICS_BIND_ADDRESS",
        "FAILOVER_PROMETHEUS_URL",
        "FAILOVER_ALERTMANAGER_URL",
        "FAILOVER_GRAFANA_PUBLIC_URL",
        "FAILOVER_MONITORING_USERNAME",
        "FAILOVER_MONITORING_PASSWORD",
        "FAILOVER_MONITORING_BEARER_TOKEN",
        "SQL_MAX_IDLE_CONNS",
        "SQL_MAX_OPEN_CONNS",
        "SQL_MAX_LIFETIME",
    )
    lines = [
        "# Managed runtime environment for api2.alltokenapi.com.",
        "# Secrets are copied directly from the alltokenapi master and never printed.",
        f"SQL_DSN={sql_dsn}",
        f"REDIS_CONN_STRING={redis_connection}",
    ]
    for key in inherited_keys:
        if key in master_env:
            lines.append(f"{key}={master_env[key]}")

    # api2 currently exposes HTTP only. A Secure cookie would make UI sessions unusable.
    lines.extend(
        (
            "SESSION_COOKIE_SECURE=false",
            "NODE_NAME=new-api-api2-slave-107-149-30-248",
            "NODE_TYPE=slave",
        )
    )

    destination = os.path.abspath(sys.argv[1])
    destination_dir = os.path.dirname(destination)
    os.makedirs(destination_dir, mode=0o700, exist_ok=True)
    file_descriptor, temporary_path = tempfile.mkstemp(
        dir=destination_dir, prefix=".new-api-slave.env.", text=True
    )
    try:
        os.fchmod(file_descriptor, 0o600)
        with os.fdopen(file_descriptor, "w", encoding="utf-8") as output:
            output.write("\n".join(lines))
            output.write("\n")
        os.replace(temporary_path, destination)
        os.chmod(destination, 0o600)
    except BaseException:
        os.close(file_descriptor)
        try:
            os.unlink(temporary_path)
        except FileNotFoundError:
            pass
        raise

    print("slave environment written")


if __name__ == "__main__":
    main()
