import { TestBed } from '@angular/core/testing';
import { DashboardLayoutService, WidgetLayoutItem } from './dashboard-layout.service';

describe('DashboardLayoutService', () => {
  let service: DashboardLayoutService;
  const testKey = 'test_dashboard_layout_key';

  const defaultWidgets: WidgetLayoutItem[] = [
    { id: 'w1', title: 'Widget 1', colSpan: 12, visible: true },
    { id: 'w2', title: 'Widget 2', colSpan: 6, visible: true },
    { id: 'w3', title: 'Widget 3', colSpan: 6, visible: true }
  ];

  beforeEach(() => {
    TestBed.configureTestingModule({});
    service = TestBed.inject(DashboardLayoutService);
    localStorage.removeItem(testKey);
  });

  afterEach(() => {
    localStorage.removeItem(testKey);
  });

  it('should return default widgets when storage is empty', () => {
    const layout = service.loadLayout(testKey, defaultWidgets);
    expect(layout.length).toBe(3);
    expect(layout[0].id).toBe('w1');
  });

  it('should save and load customized layout', () => {
    const custom: WidgetLayoutItem[] = [
      { id: 'w2', title: 'Widget 2', colSpan: 12, visible: true },
      { id: 'w1', title: 'Widget 1', colSpan: 4, visible: false },
      { id: 'w3', title: 'Widget 3', colSpan: 8, visible: true }
    ];

    service.saveLayout(testKey, custom);
    const loaded = service.loadLayout(testKey, defaultWidgets);

    expect(loaded[0].id).toBe('w2');
    expect(loaded[0].colSpan).toBe(12);
    expect(loaded[1].id).toBe('w1');
    expect(loaded[1].visible).toBe(false);
  });

  it('should merge new widgets if defaults change', () => {
    const custom: WidgetLayoutItem[] = [
      { id: 'w1', title: 'Widget 1', colSpan: 6, visible: true }
    ];
    service.saveLayout(testKey, custom);

    const loaded = service.loadLayout(testKey, defaultWidgets);
    expect(loaded.length).toBe(3);
    expect(loaded[0].id).toBe('w1');
    expect(loaded[1].id).toBe('w2');
    expect(loaded[2].id).toBe('w3');
  });

  it('should reset layout and remove key from localStorage', () => {
    service.saveLayout(testKey, [{ id: 'w2', title: 'W2', colSpan: 12, visible: true }]);
    const reset = service.resetLayout(testKey, defaultWidgets);

    expect(reset.length).toBe(3);
    expect(reset[0].id).toBe('w1');
    expect(localStorage.getItem(testKey)).toBeNull();
  });
});
