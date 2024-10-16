import { Component, EventEmitter, Input, Output } from '@angular/core';
import { MetadataObject } from '../services/explore.service';
import { TableComponent } from './table/table.component';
import { TreemapComponent } from './treemap/treemap.component';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';

@Component({
  selector: 'app-viewer',
  standalone: true,
  imports: [TableComponent, TreemapComponent, MatCardModule, MatIconModule],
  templateUrl: './viewer.component.html',
  styleUrl: './viewer.component.css',
})
export class ViewerComponent {
  @Input({ required: true }) directoryList$!: MetadataObject[];
  @Input({ required: true }) view!: string;
  @Input({ required: true }) currentPath!: string;

  @Output() newPathEvent = new EventEmitter<string>();
  @Output() onBackEvent = new EventEmitter();
  @Output() onRefreshEvent = new EventEmitter();

  goTo(path: string) {
    this.newPathEvent.emit(path);
  }

  goBack() {
    this.onBackEvent.emit();
  }

  onRefresh() {
    this.onRefreshEvent.emit();
  }
}
