import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  LucidePlus,
  LucideDatabase,
  LucideLayers
} from '@lucide/angular';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { TechLogoComponent, TechIconKey } from '@features/admin/templates/components/tech-logo/tech-logo.component';
import { ApprovedTemplate } from '../../step-ide-environment.component';

export interface TechBadgeItem {
  label: string;
  icon: TechIconKey;
}

@Component({
  selector: 'ide-template-selector',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucidePlus,
    LucideDatabase,
    LucideLayers,
    ComboboxComponent,
    TechLogoComponent
  ],
  templateUrl: './ide-template-selector.component.html',
  styleUrl: './ide-template-selector.component.scss'
})
export class IdeTemplateSelectorComponent {
  approvedTemplates = input<ApprovedTemplate[]>([]);
  selectedTemplate = input<ApprovedTemplate | null>(null);
  templateOptions = input<ComboboxOption[]>([]);

  templateSelected = output<string>();
  requestTemplate = output<void>();

  onTemplateOptionSelected(opt: ComboboxOption): void {
    if (opt?.value) {
      this.templateSelected.emit(opt.value);
    }
  }

  deriveTechIcon(tpl: ApprovedTemplate): TechIconKey {
    const name = (tpl.name || '').toLowerCase();
    const image = (tpl.docker_image || '').toLowerCase();
    if (name.includes('python') || image.includes('python')) return 'python';
    if (name.includes('typescript') || image.includes('typescript')) return 'typescript';
    if (name.includes('nest') || image.includes('nest')) return 'node';
    if (name.includes('node') || name.includes('javascript') || image.includes('node')) return 'node';
    if (name.includes('react') || image.includes('react')) return 'react';
    if (name.includes('angular') || image.includes('angular')) return 'angular';
    if (name.includes('vue') || image.includes('vue')) return 'vue';
    if (name.includes('c++') || name.includes('cpp')) return 'cpp';
    if (name.includes('c/') || name === 'c' || image.includes('gcc')) return 'c';
    if (name.includes('java') || image.includes('java')) return 'java';
    if (name.includes('go') || image.includes('golang')) return 'go';
    if (name.includes('rust') || image.includes('rust')) return 'rust';
    if (name.includes('c#') || name.includes('csharp')) return 'dotnet';
    return 'docker';
  }

  deriveTechBadges(tpl: ApprovedTemplate): TechBadgeItem[] {
    const name = (tpl.name || '').toLowerCase();
    const badges: TechBadgeItem[] = [];

    if (name.includes('python')) badges.push({ label: 'Python', icon: 'python' });
    if (name.includes('data science') || name.includes('ia') || name.includes('analytics')) {
      badges.push({ label: 'Pandas', icon: 'pandas' });
      badges.push({ label: 'NumPy', icon: 'numpy' });
    }
    if (name.includes('node') || name.includes('javascript')) badges.push({ label: 'Node.js', icon: 'node' });
    if (name.includes('nest')) badges.push({ label: 'NestJS', icon: 'node' });
    if (name.includes('react')) badges.push({ label: 'React', icon: 'react' });
    if (name.includes('angular')) badges.push({ label: 'Angular', icon: 'angular' });
    if (name.includes('java')) badges.push({ label: 'Java', icon: 'java' });
    if (name.includes('spring')) badges.push({ label: 'Spring Boot', icon: 'spring' });
    if (name.includes('c++') || name.includes('cpp')) badges.push({ label: 'C++', icon: 'cpp' });
    if (name.includes('c/') || name.includes('concurrencia') || name.includes('operativos')) badges.push({ label: 'GCC / Linux', icon: 'c' });
    if (name.includes('postgres') || name.includes('sql')) badges.push({ label: 'PostgreSQL', icon: 'postgres' });

    return badges.length > 0 ? badges : [{ label: tpl.name, icon: this.deriveTechIcon(tpl) }];
  }
}
