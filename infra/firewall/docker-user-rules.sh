#!/usr/bin/env bash
set -euo pipefail

if [ "$EUID" -ne 0 ]; then
    echo "Error: Este script debe ser ejecutado con privilegios de root (sudo)." >&2
    exit 1
fi

# Puertos sensibles a bloquear
PORTS=(5432 9090 3000 8080)

# Identificar interfaz de salida hacia internet
DEFAULT_OUT_IFACE=$(ip route show default 2>/dev/null | awk '{for(i=1;i<=NF;i++) if($i=="dev") print $(i+1)}' | head -n1)
OUT_IFACE="${OUT_IFACE:-${DEFAULT_OUT_IFACE:-eth0}}"
echo "=> Interfaz de salida detectada: $OUT_IFACE"

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
# No se bloquea la salida de docker0 a internet para permitir descargas de paquetes (npm/pip) en builds y pruebas.
# El aislamiento de exámenes sin salida se aplica por etiqueta mediante solv.exam.no-egress.

for port in "${PORTS[@]}"; do
    apply_rule DOCKER-USER -i "$INTERFACE_DEFAULT" -d "$GATEWAY_DEFAULT" -p tcp --dport "$port" -j DROP
done

if [ -n "$INTERFACE_SOLV" ] && [ -n "$GATEWAY_SOLV" ]; then
    echo "Configurando bloqueos para la red personalizada ($INTERFACE_SOLV)..."
    # Bloquear tráfico entrante directo desde el exterior hacia solv_net (Zero Trust)
    apply_rule DOCKER-USER -i "$OUT_IFACE" -o "$INTERFACE_SOLV" -j DROP
    for port in "${PORTS[@]}"; do
        apply_rule DOCKER-USER -i "$INTERFACE_SOLV" -d "$GATEWAY_SOLV" -p tcp --dport "$port" -j DROP
    done
fi

# Defensa adicional del examen sin salida. La red interna no tiene puerta de
# enlace y ya corta el tráfico, pero si el contenedor recibiera una segunda red
# con gateway esta regla lo deja sin salida igual. La lista se deriva de la
# etiqueta que pone el backend al crear el entorno, no de una lista de IPs a mano.
if command -v docker &>/dev/null; then
    echo "Configurando defensa de contenedores de examen etiquetados..."
    EXAM_CONTAINERS=$(docker ps -q --filter "label=solv.exam.no-egress=true" 2>/dev/null || true)
    for cid in $EXAM_CONTAINERS; do
        EXAM_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}} {{end}}' "$cid" 2>/dev/null | awk '{print $1}')
        if [ -z "$EXAM_IP" ]; then
            continue
        fi
        echo "=> Contenedor de examen $cid ($EXAM_IP) sin salida"
        apply_rule DOCKER-USER -s "$EXAM_IP" -o "$OUT_IFACE" ! -d 10.0.0.0/8 -j DROP
        apply_rule DOCKER-USER -s "$EXAM_IP" -o "$OUT_IFACE" ! -d 172.16.0.0/12 -j DROP
        apply_rule DOCKER-USER -s "$EXAM_IP" -o "$OUT_IFACE" ! -d 192.168.0.0/16 -j DROP
    done
fi

echo "=== Reglas de firewall aplicadas exitosamente ==="
