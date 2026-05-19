import { bootstrapApplication, BootstrapContext } from '@angular/platform-browser';
import { AppComponent } from './app/app';
import { appConfigServer } from './app/app.config.server';

const bootstrap = (context: BootstrapContext) =>
	bootstrapApplication(AppComponent, appConfigServer, context);

export default bootstrap;
