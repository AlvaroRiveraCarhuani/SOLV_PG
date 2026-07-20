import { Component, signal, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';

@Component({
  selector: 'app-root',
  templateUrl: './app.html',
  styleUrl: './app.css'
})
export class App {
  private http = inject(HttpClient);
  
  protected loading = signal(false);
  protected labUrl = signal<string | null>(null);
  protected errorMsg = signal<string | null>(null);

  startLab() {
    this.loading.set(true);
    this.errorMsg.set(null);
    this.labUrl.set(null);

    const payload = {
      user_id: "2757773f-58d7-4a2f-af05-d224be746aee",
      template_id: "1281b9f1-3d25-4d34-b18d-a8c9faab20e9",
      ram_limit_mb: 256,
      user_email: "edwin@uab.edu.bo"
    };

    this.http.post<any>('http://localhost:3000/labs/start', payload).subscribe({
      next: (res) => {
        const prefix = payload.user_email.split('@')[0];
        this.labUrl.set(`http://${prefix}-lab.solv.uab.edu.bo`);
        this.loading.set(false);
      },
      error: (err) => {
        console.error(err);
        this.errorMsg.set('Fallo de conexión. Verifica que el backend de Go esté corriendo en el puerto 3000 y tenga CORS habilitado.');
        this.loading.set(false);
      }
    });
  }
}
