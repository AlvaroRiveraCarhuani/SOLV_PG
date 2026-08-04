#!/usr/bin/env bash
set -euo pipefail

if [ "$EUID" -ne 0 ]; then
    echo "Error: Este script debe ser ejecutado con privilegios de root (sudo)." >&2
    exit 1
fi

# Puertos sensibles a bloquear
PORTS=(5432 9090 3000 8080)

INTERFACE_DEFAULT="docker0"
GATEWAY_DEFAULT="172.17.0.1"

# Identificar la interfaz de la red personalizada solv_net
INTERFACE_SOLV=""
GATEWAY_SOLV=""

if command -v docker &>/dev/null; then
    NET_ID=$(docker network inspect solv_net -f '{{.Id}}' 2>/dev/null | cut -c1-12 || true)
    if [ -n "$NET_ID" ]; then
        INTERFACE_SOLV="br-$NET_ID"
        GATEWAY_SOLV=$(docker network inspect solv_net -f '{{(index .IPAM.Config 0).Gateway}}' 2>/dev/null || true)
        echo "=> Detectada interfaz para solv_net: $INTERFACE_SOLV (Gateway: $GATEWAY_SOLV)"
    else
        echo "=> La red personalizada solv_net no está creada todavía. Se aplicará solo a docker0."
    fi
fi

# Función para aplicar reglas de manera idempotente insertando al inicio (-I)
apply_rule() {
    local rule=("$@")
    # Verificar si la regla existe
    if ! iptables -C "${rule[@]}" 2>/dev/null; then
        echo "=> Aplicando regla: iptables -I ${rule[*]}"
        iptables -I "${rule[@]}"
    else
        echo "=> Regla ya existente (omitida): iptables ${rule[*]}"
    fi
}

echo "=== Configurando reglas de firewall Zero Trust en DOCKER-USER ==="

echo "Configurando bloqueos para la red por defecto ($INTERFACE_DEFAULT)..."
apply_rule DOCKER-USER -i "$INTERFACE_DEFAULT" -o eth0 -j DROP

for port in "${PORTS[@]}"; do
    apply_rule DOCKER-USER -i "$INTERFACE_DEFAULT" -d "$GATEWAY_DEFAULT" -p tcp --dport "$port" -j DROP
done

if [ -n "$INTERFACE_SOLV" ] && [ -n "$GATEWAY_SOLV" ]; then
    echo "Configurando bloqueos para la red personalizada ($INTERFACE_SOLV)..."
    apply_rule DOCKER-USER -i "$INTERFACE_SOLV" -o eth0 -j DROP
    for port in "${PORTS[@]}"; do
        apply_rule DOCKER-USER -i "$INTERFACE_SOLV" -d "$GATEWAY_SOLV" -p tcp --dport "$port" -j DROP
    done
fi

echo "=== Reglas de firewall aplicadas exitosamente ==="
