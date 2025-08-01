/**
 * Copyright 2023 Jack Bister
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { JSX, h } from "preact";
import {
  Anchor,
  AppShell,
  Group,
  MantineProvider,
  Text,
  ThemeIcon,
  Title,
  UnstyledButton,
  useMantineColorScheme,
} from "@mantine/core";
import { IconSearch, IconSettings } from "@tabler/icons-preact";

import '@mantine/core/styles.css';

interface MainLinkProps {
  iconSvg: JSX.Element;
  href: string;
  label: string;
}

const MainLink = ({ iconSvg, href, label }: MainLinkProps) => {
  const { colorScheme } = useMantineColorScheme();
  return (
    <UnstyledButton
      component="a"
      href={href}
      style={(theme) => ({
        display: "block",
        width: "100%",
        padding: theme.spacing.xs,
        borderRadius: theme.radius.sm,
        color:
          colorScheme === "dark" ? theme.colors.dark[0] : theme.black,

        "&:hover": {
          backgroundColor:
            colorScheme === "dark"
              ? theme.colors.dark[6]
              : theme.colors.gray[0],
        },
      })}
    >
      <Group>
        <ThemeIcon variant="light">{iconSvg}</ThemeIcon>
        <Text size="sm">{label}</Text>
      </Group>
    </UnstyledButton>
  );
};

export const LogsuckAppShell = (props: any) => {
  return (
    <MantineProvider>
      <div id="app">
        <AppShell
          padding="md"
          navbar={
            { width: 160, breakpoint: 'sm' }
          }
          styles={(theme: any) => ({
            main: {
              backgroundColor:
                theme.colorScheme === "dark"
                  ? theme.colors.dark[8]
                  : theme.colors.gray[0],
            },
          })}
        >
          <AppShell.Navbar>
            <AppShell.Section>
              <Anchor href="/">
                <Title px={8}>
                  Logsuck
                </Title>
              </Anchor>
            </AppShell.Section>
            <AppShell.Section>
              <MainLink
                label="Search"
                iconSvg={<IconSearch />}
                href="/search"
              ></MainLink>
              <MainLink
                label="Config"
                iconSvg={<IconSettings />}
                href="/config"
              ></MainLink>
            </AppShell.Section>
          </AppShell.Navbar>
          <AppShell.Main>
            {props.children}
          </AppShell.Main>
        </AppShell>
      </div>
    </MantineProvider >
  );
};
