# Manual Operativo — Slice 10: TLS Wildcard & Operación desec.io

## 1. Obtención y Configuración del Token desec.io

1. Acceda al panel de administración de [desec.io](https://desec.io).
2. En la sección **Tokens**, cree un token dedicado de API denominado `traefik-acme`.
3. Copie el valor del secreto generado y colóquelo exclusivamente en el archivo `.env` del servidor:
   ```bash
   DESEC_TOKEN=z92raAihVX6NPrLDPWaSj1GYEeiq
   DESEC_DOMAIN=solv.dedyn.io
   COOKIE_DOMAIN=.solv.dedyn.io
   ```
4. **Advertencia de Seguridad:** El token `DESEC_TOKEN` jamás debe ser incluido en repositorios ni mensajes de commit.

---

## 2. Aprovisionamiento Automático de Registros DNS

Para inicializar o actualizar los registros A (apex) y CNAME (wildcard) en desec.io apuntando a la IP pública del servidor:

```bash
chmod +x infra/desec/setup_dns.sh
./infra/desec/setup_dns.sh
```

El script detecta la IP pública del servidor mediante `ifconfig.me` y configura idempotentemente:
- `solv.dedyn.io` A -> IP pública
- `*.solv.dedyn.io` CNAME -> `solv.dedyn.io.`

---

## 3. Verificación de Certificados TLS

### Verificación Manual con OpenSSL
Para comprobar la terminación TLS y la validez del certificado devuelto por Traefik:

```bash
openssl s_client -connect solv.dedyn.io:443 -servername solv.dedyn.io
```

### Verificación vía Métrica Prometheus
La plataforma expone la vigencia restante del certificado en días mediante la métrica gauge:

```bash
curl -s http://localhost:3000/metrics | grep solv_cert_expiry_days
```
