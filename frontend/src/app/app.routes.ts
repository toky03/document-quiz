import { Routes } from '@angular/router';
import { UploadComponent } from './components/upload/upload.component';
import { QuizComponent } from './components/quiz/quiz.component';

export const routes: Routes = [
  { path: '', component: UploadComponent },
  { path: 'quiz/:chapterId', component: QuizComponent },
  { path: '**', redirectTo: '' }
];
