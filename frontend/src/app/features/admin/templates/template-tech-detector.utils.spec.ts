import { detectTemplateTech, inferTemplateTechnologies } from './template-tech-detector.utils';

describe('template tech inference', () => {
  it('prioriza el runtime declarado en stack_config sobre el texto', () => {
    const tech = detectTemplateTech({
      name: 'Laboratorio Especializado',
      docker_image: 'custom/runner:1.0',
      stack_config: {
        runtime_id: 'rust-178',
        web_platform_id: 'node-20'
      }
    });

    expect(tech.iconType).toBe('rust');
  });

  it('no confunde JavaScript con Java ni la letra c de "stack" con C', () => {
    const tech = detectTemplateTech({
      name: 'Entorno Full-Stack JS',
      docker_image: 'solv-local/stack:660620ce5745'
    });

    expect(tech.iconType).toBe('node');
    expect(tech.iconType).not.toBe('java');
    expect(tech.iconType).not.toBe('c');
  });

  it('devuelve todas las tecnologías del texto en orden de lectura', () => {
    const techs = inferTemplateTechnologies({
      name: 'Entorno Full-Stack JS',
      docker_image: 'solv-local/stack:660620ce5745',
      description: 'Node.js, Next.js y PostgreSQL para clases académicas'
    });

    expect(techs.map(tech => tech.id)).toEqual(['node', 'nextjs', 'postgres']);
  });

  it('reconoce Java con Spring Boot, Maven y Gradle sin duplicar el lenguaje', () => {
    const techs = inferTemplateTechnologies({
      name: 'Java 21 Enterprise & Spring Boot',
      docker_image: 'eclipse-temurin:21-jdk',
      description: 'JDK 21 Temurin con Maven y Gradle'
    });

    expect(techs.map(tech => tech.id)).toEqual(['java', 'spring', 'maven', 'gradle']);
  });

  it('no agrega C cuando el entorno es C++', () => {
    const techs = inferTemplateTechnologies({
      name: 'C/C++ Sistemas Operativos',
      docker_image: 'gcc:13',
      description: 'Compiladores GCC y Clang'
    });

    expect(techs.map(tech => tech.id)).toEqual(['cpp', 'gcc']);
  });
});
