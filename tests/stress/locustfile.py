import uuid
import random
# pyrefly: ignore [missing-import]
from locust import HttpUser, task, between, events
# pyrefly: ignore [missing-import]
from locust.exception import StopUser

class EstudianteSOLV(HttpUser):
    wait_time = between(1, 3)

    def on_start(self):
        self.user_id = str(uuid.uuid4())
        self.prefix = f"estudiante.{random.randint(1000, 99999)}"
        self.user_email = f"{self.prefix}@uab.edu.bo"
        
        self.template_id = "123e4567-e89b-12d3-a456-426614174000"

    @task
    def iniciar_laboratorio(self):
        payload = {
            "user_id": self.user_id,
            "template_id": self.template_id,
            "ram_limit_mb": 100,
            "user_email": self.user_email
        }

        with self.client.post("/labs/start", json=payload, catch_response=True, name="/labs/start") as response:
            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"Fallo con código {response.status_code}: {response.text}")
        
        raise StopUser()

@events.test_start.add_listener
def on_test_start(environment, **kwargs):
    print("🚀 Iniciando prueba de estrés de SOLV. Generando carga concurrente...")

@events.test_stop.add_listener
def on_test_stop(environment, **kwargs):
    print("🛑 Prueba finalizada. Revisa la interfaz web para exportar el CSV de métricas.")
