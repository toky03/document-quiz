import { Routes } from '@angular/router';
import { UploadComponent } from './components/upload/upload.component';
import { QuizComponent } from './components/quiz/quiz.component';
import { DisasterComponent } from './components/disaster/disaster.component';

export const routes: Routes = [
  { path: '', component: UploadComponent },
  { path: 'quiz/:chapterId', component: QuizComponent },
  { path: 'disaster', component: DisasterComponent },
  { path: '**', redirectTo: '' }
];
