import * as vscode from 'vscode';
import { SolvClient } from './client/solv-client';
import { CommentDecorator } from './comments/comment-decorator';
import { CommentsController } from './comments/comments-controller';
import { SolvSettings } from './config/solv-settings';
import { KeystrokeCapturer } from './telemetry/keystroke-capturer';

export function activate(context: vscode.ExtensionContext): void {
  const settings = new SolvSettings(context);
  const client = new SolvClient(settings);
  const decorator = new CommentDecorator();
  const comments = new CommentsController(client, decorator, settings);
  const telemetry = new KeystrokeCapturer(client, settings);

  context.subscriptions.push(
    decorator,
    comments,
    telemetry,
    vscode.commands.registerCommand('solv.configure', async () => {
      if (await settings.configure()) {
        await comments.refresh(true);
      }
    }),
    vscode.commands.registerCommand('solv.refreshComments', () => comments.refresh(true)),
    vscode.commands.registerCommand('solv.showTelemetryStatus', () => telemetry.showStatus())
  );

  telemetry.start();
  void comments.refresh();
}

export function deactivate(): void {
  // Los recursos se liberan por context.subscriptions.
}
